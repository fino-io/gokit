package tracing

import (
	"context"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var grpcPropagator = propagation.NewCompositeTextMapPropagator(
	propagation.TraceContext{},
	propagation.Baggage{},
)

// extractGRPCContext extracts OpenTelemetry propagation metadata from md.
func extractGRPCContext(ctx context.Context, md metadata.MD) context.Context {
	if trace.SpanContextFromContext(ctx).IsValid() {
		return ctx
	}
	return grpcPropagator.Extract(ctx, metadataTextMap(md))
}

// GRPCUnaryClientInterceptor creates a client span and propagates its context
// to unary gRPC calls.
func GRPCUnaryClientInterceptor(tracer trace.Tracer) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if tracer == nil {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		ctx, span := tracer.Start(ctx, method, trace.WithSpanKind(trace.SpanKindClient))
		defer span.End()

		md, _ := metadata.FromOutgoingContext(ctx)
		md = md.Copy()
		grpcPropagator.Inject(ctx, metadataTextMap(md))
		ctx = metadata.NewOutgoingContext(ctx, md)

		if err := invoker(ctx, method, req, reply, cc, opts...); err != nil {
			recordError(span, err)
			return err
		}
		return nil
	}
}

// metadataTextMap implements the TextMapCarrier interface for gRPC metadata.
type metadataTextMap metadata.MD

// Get retrieves a single value for a given key.
func (m metadataTextMap) Get(key string) string {
	if values := m[key]; len(values) > 0 {
		return values[0]
	}
	return ""
}

// Set sets a value for a given key.
func (m metadataTextMap) Set(key string, value string) {
	m[key] = []string{value}
}

// Keys lists the keys stored in this carrier.
func (m metadataTextMap) Keys() []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
