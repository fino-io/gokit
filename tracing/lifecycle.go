package tracing

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// ShutdownFunc releases tracing resources. It is always safe to call.
type ShutdownFunc func(context.Context) error

func noopShutdown(context.Context) error {
	return nil
}

func noopTracer(name string) trace.Tracer {
	return noop.NewTracerProvider().Tracer(name)
}
