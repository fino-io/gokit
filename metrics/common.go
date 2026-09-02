package metrics

import (
	"context"
	"errors"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
)

var defaultBuckets = prometheus.DefBuckets

func registerOrReuse[T prometheus.Collector](registerer prometheus.Registerer, collector T) T {
	if err := registerer.Register(collector); err != nil {
		var alreadyRegistered prometheus.AlreadyRegisteredError
		if errors.As(err, &alreadyRegistered) {
			existing, ok := alreadyRegistered.ExistingCollector.(T)
			if ok {
				return existing
			}
		}
		panic(err)
	}
	return collector
}

func result(ctx context.Context, err error) string {
	if ctx != nil {
		switch ctx.Err() {
		case context.DeadlineExceeded:
			return "timeout"
		case context.Canceled:
			return "canceled"
		}
	}
	if err != nil {
		return "failure"
	}
	return "success"
}

func httpResult(ctx context.Context, response *http.Response, err error) string {
	if value := result(ctx, err); value != "success" {
		return value
	}
	if response == nil {
		return "failure"
	}
	switch {
	case response.StatusCode >= http.StatusInternalServerError:
		return "server_error"
	case response.StatusCode >= http.StatusBadRequest:
		return "client_error"
	default:
		return "success"
	}
}
