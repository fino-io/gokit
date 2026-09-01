package accesslog

import (
	"net/http"
	"time"

	"github.com/felixge/httpsnoop"
)

// HTTPMiddleware logs completed HTTP server requests.
func HTTPMiddleware(cfg Config) func(http.Handler) http.Handler {
	return httpMiddleware(cfg, defaultLog)
}

func httpMiddleware(cfg Config, log logFunc) func(http.Handler) http.Handler {
	policy := newPolicy(cfg, cfg.HTTP.SkipPaths)
	return func(next http.Handler) http.Handler {
		if next == nil {
			next = http.NotFoundHandler()
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			metrics := httpsnoop.Metrics{Code: http.StatusOK}
			defer func() {
				recovered := recover()
				if recovered != nil {
					metrics.Code = http.StatusInternalServerError
				}
				logHTTP(r, metrics, time.Since(started), policy, log)
				if recovered != nil {
					panic(recovered)
				}
			}()
			metrics.CaptureMetrics(w, func(w http.ResponseWriter) {
				next.ServeHTTP(w, r)
			})
		})
	}
}

func logHTTP(r *http.Request, metrics httpsnoop.Metrics, duration time.Duration, policy *policy, log logFunc) {
	requestID := r.Header.Get("X-Request-ID")
	if !policy.shouldLog(r.URL.Path, requestID, duration, importantHTTPStatus(metrics.Code)) {
		return
	}
	fields := []any{
		"protocol", "http",
		"method", r.Method,
		"path", r.URL.Path,
		"status", metrics.Code,
		"bytes", metrics.Written,
		"duration_ms", float64(duration.Microseconds()) / 1000,
		"remote_ip", remoteHost(r.RemoteAddr),
		"request_id", requestID,
	}
	log(r.Context(), httpLevel(metrics.Code), "http access", fields...)
}

func importantHTTPStatus(code int) bool {
	if code >= http.StatusInternalServerError {
		return true
	}
	switch code {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests:
		return true
	default:
		return false
	}
}

func httpLevel(code int) level {
	switch {
	case code >= http.StatusInternalServerError:
		return levelError
	case code >= http.StatusBadRequest:
		return levelWarn
	default:
		return levelInfo
	}
}
