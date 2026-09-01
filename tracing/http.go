package tracing

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var httpPropagator = propagation.NewCompositeTextMapPropagator(
	propagation.TraceContext{},
	propagation.Baggage{},
)

// HTTPToContext extracts OpenTelemetry propagation headers from req.
func HTTPToContext(ctx context.Context, req *http.Request) context.Context {
	if req == nil || trace.SpanContextFromContext(ctx).IsValid() {
		return ctx
	}
	return httpPropagator.Extract(ctx, propagation.HeaderCarrier(req.Header))
}

// HTTPAttributes returns low-cardinality HTTP attributes for a request span.
func HTTPAttributes(req *http.Request) []attribute.KeyValue {
	if req == nil {
		return nil
	}

	attrs := []attribute.KeyValue{
		attribute.String("http.method", req.Method),
	}
	if req.URL == nil {
		return attrs
	}

	attrs = append(attrs,
		attribute.String("http.url", req.URL.String()),
		attribute.String("http.scheme", httpScheme(req)),
		attribute.String("http.path", req.URL.Path),
	)
	if req.URL.RawQuery != "" {
		attrs = append(attrs, attribute.String("http.query", req.URL.RawQuery))
	}

	return attrs
}

func InjectHTTPHeader(ctx context.Context, header http.Header) {
	if header == nil {
		return
	}
	httpPropagator.Inject(ctx, propagation.HeaderCarrier(header))
}

func httpScheme(req *http.Request) string {
	if req.URL != nil && req.URL.Scheme != "" {
		return req.URL.Scheme
	}
	if req.TLS != nil {
		return "https"
	}
	return "http"
}
