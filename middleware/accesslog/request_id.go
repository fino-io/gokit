package accesslog

import (
	"context"
	"strings"

	"github.com/fino-io/finokit/logs"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

const (
	requestIDHeader      = "X-Request-ID"
	requestIDMetadataKey = "x-request-id"
	maxRequestIDLength   = 128
)

type requestIDContextKey struct{}

func ensureRequestID(value string) string {
	value = strings.TrimSpace(value)
	if validRequestID(value) {
		return value
	}

	return uuid.NewString()
}

func withRequestID(ctx context.Context, requestID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = context.WithValue(ctx, requestIDContextKey{}, requestID)
	return logs.WithFields(ctx, logs.Field{Key: "request_id", Value: requestID})
}

func requestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

func requestIDFromIncomingMetadata(ctx context.Context) string {
	incoming, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := incoming.Get(requestIDMetadataKey)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func validRequestID(requestID string) bool {
	if requestID == "" || len(requestID) > maxRequestIDLength {
		return false
	}
	for i := 0; i < len(requestID); i++ {
		if requestID[i] < 0x21 || requestID[i] > 0x7e {
			return false
		}
	}
	return true
}
