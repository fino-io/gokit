package metrics

import (
	"net/http"

	"github.com/felixge/httpsnoop"
)

type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	tracked := &responseWriter{status: http.StatusOK}
	tracked.ResponseWriter = httpsnoop.Wrap(w, httpsnoop.Hooks{
		WriteHeader: func(next httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
			return func(status int) {
				if !tracked.wroteHeader {
					tracked.status = status
					tracked.wroteHeader = true
				}
				next(status)
			}
		},
	})
	return tracked
}

func (w *responseWriter) StatusCode() int {
	return w.status
}
