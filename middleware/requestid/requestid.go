// Package requestid propagates request IDs across HTTP and gRPC boundaries.
package requestid

import (
	"context"
	"net/http"
	"strings"

	"github.com/fino-io/finokit/logs"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	// Header is the HTTP header used to carry a request ID.
	Header = "X-Request-ID"
	// MetadataKey is the gRPC metadata key used to carry a request ID.
	MetadataKey = "x-request-id"
	maxLength   = 128
)

type contextKey struct{}

// Ensure returns value when it is a safe request ID, or creates a UUID when it
// is missing or invalid.
func Ensure(value string) string {
	value = strings.TrimSpace(value)
	if valid(value) {
		return value
	}
	return uuid.NewString()
}

// WithContext associates requestID with ctx and adds it to the structured log
// fields carried by the context.
func WithContext(ctx context.Context, requestID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = context.WithValue(ctx, contextKey{}, requestID)
	return logs.WithFields(ctx, logs.Field{Key: "request_id", Value: requestID})
}

// FromContext returns the request ID stored in ctx.
func FromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	requestID, _ := ctx.Value(contextKey{}).(string)
	return requestID
}

// FromIncomingMetadata returns the first request ID from incoming gRPC
// metadata.
func FromIncomingMetadata(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	incoming, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := incoming.Get(MetadataKey)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// HTTPMiddleware assigns a request ID, propagates it through the request
// context, and returns it in the response header.
func HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if next == nil {
			next = http.NotFoundHandler()
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := Ensure(r.Header.Get(Header))
			r.Header.Set(Header, requestID)
			w.Header().Set(Header, requestID)
			next.ServeHTTP(w, r.WithContext(WithContext(r.Context(), requestID)))
		})
	}
}

// UnaryClientInterceptor propagates a request ID to outgoing unary gRPC calls.
func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		requestID := FromContext(ctx)
		if requestID == "" {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		md, _ := metadata.FromOutgoingContext(ctx)
		md = md.Copy()
		md.Set(MetadataKey, requestID)
		return invoker(metadata.NewOutgoingContext(ctx, md), method, req, reply, cc, opts...)
	}
}

// UnaryServerInterceptor assigns a request ID and stores it in the request
// context for downstream handlers.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		requestID := Ensure(FromIncomingMetadata(ctx))
		return handler(WithContext(ctx, requestID), req)
	}
}

func valid(requestID string) bool {
	if requestID == "" || len(requestID) > maxLength {
		return false
	}
	for i := 0; i < len(requestID); i++ {
		if requestID[i] < 0x21 || requestID[i] > 0x7e {
			return false
		}
	}
	return true
}
