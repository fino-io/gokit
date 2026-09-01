package tracing

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// HTTPServerMiddleware extracts the incoming trace context and creates the
// transport-level server span before the handler runs.
//
// The middleware should wrap access logging and routing so those operations
// can use the same trace context. A nil tracer disables local span creation
// while preserving any incoming remote trace context.
func HTTPServerMiddleware(tracer trace.Tracer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if next == nil {
			next = http.NotFoundHandler()
		}

		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := HTTPToContext(request.Context(), request)
			ctx, span := startServerSpan(
				tracer,
				ctx,
				"HTTP "+request.Method,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(HTTPAttributes(request)...),
			)
			defer span.End()

			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

// GRPCUnaryServerTracingInterceptor extracts the incoming trace context and
// creates the transport-level server span before other unary interceptors run.
// It should be installed before access logging and endpoint handlers.
//
// A nil tracer disables local span creation while preserving any incoming
// remote trace context.
func GRPCUnaryServerTracingInterceptor(tracer trace.Tracer) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, request any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, _ := metadata.FromIncomingContext(ctx)
		ctx = GRPCToContext(ctx, md)

		name := "gRPC"
		if info != nil && info.FullMethod != "" {
			name = info.FullMethod
		}

		ctx, span := startServerSpan(tracer, ctx, name, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()

		response, err := handler(ctx, request)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return response, err
	}
}

func startServerSpan(tracer trace.Tracer, ctx context.Context, name string, options ...trace.SpanStartOption) (context.Context, trace.Span) {
	if tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	return tracer.Start(ctx, name, options...)
}
