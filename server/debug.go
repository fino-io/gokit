package server

import (
	"context"
	"net"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/etherlabsio/healthcheck/v2"
)

func NewDebugHandler(readinessAddr string, metricsHandler http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	mux.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	mux.Handle("/metrics", metricsHandler)
	mux.HandleFunc("/livez", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	readyz := healthcheck.Handler(
		healthcheck.WithTimeout(5*time.Second),
		healthcheck.WithChecker("http", healthcheck.CheckerFunc(func(ctx context.Context) error {
			dialer := &net.Dialer{Timeout: time.Second}
			conn, err := dialer.DialContext(ctx, "tcp", readinessAddr)
			if err != nil {
				return err
			}
			return conn.Close()
		})),
	)
	mux.Handle("/readyz", readyz)
	mux.Handle("/healthz", readyz)
	return mux
}
