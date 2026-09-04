package tracing

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

func New(name string) (trace.Tracer, ShutdownFunc, error) {
	return newTracer(context.Background(), name, loadConfig())
}
