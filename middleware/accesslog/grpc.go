package accesslog

import (
	"context"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// UnaryClientInterceptor propagates the request ID to outgoing unary gRPC calls.
func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		requestID := requestIDFromContext(ctx)
		if requestID == "" {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		md, _ := metadata.FromOutgoingContext(ctx)
		md = md.Copy()
		md.Set(requestIDMetadataKey, requestID)
		return invoker(metadata.NewOutgoingContext(ctx, md), method, req, reply, cc, opts...)
	}
}

// UnaryServerInterceptor assigns a request ID and logs completed unary gRPC server calls.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return unaryServerInterceptor(loadConfig(), defaultLog)
}

func unaryServerInterceptor(cfg config, log logFunc) grpc.UnaryServerInterceptor {
	policy := newPolicy(cfg, cfg.GRPC.SkipMethods)
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (response any, err error) {
		requestID := ensureRequestID(requestIDFromIncomingMetadata(ctx))
		ctx = withRequestID(ctx, requestID)

		started := time.Now()
		defer func() {
			recovered := recover()
			code := status.Code(err)
			if recovered != nil {
				code = codes.Internal
			}
			logGRPC(ctx, info.FullMethod, requestID, code, time.Since(started), policy, log)
			if recovered != nil {
				panic(recovered)
			}
		}()
		return handler(ctx, req)
	}
}

func logGRPC(ctx context.Context, method, requestID string, code codes.Code, duration time.Duration, policy *policy, log logFunc) {
	if !policy.shouldLog(method, requestID, duration, importantGRPCCode(code)) {
		return
	}
	fields := []any{
		"protocol", "grpc",
		"method", method,
		"code", code.String(),
		"duration_ms", float64(duration.Microseconds()) / 1000,
		"remote_ip", grpcRemoteIP(ctx),
		"request_id", requestID,
	}
	log(ctx, grpcLevel(code), "grpc access", fields...)
}

func grpcRemoteIP(ctx context.Context) string {
	remote, ok := peer.FromContext(ctx)
	if !ok || remote.Addr == nil {
		return ""
	}
	if address, ok := remote.Addr.(*net.TCPAddr); ok {
		return address.IP.String()
	}
	return remoteHost(remote.Addr.String())
}

func importantGRPCCode(code codes.Code) bool {
	switch code {
	case codes.Unknown,
		codes.DeadlineExceeded,
		codes.PermissionDenied,
		codes.ResourceExhausted,
		codes.Unavailable,
		codes.Unauthenticated,
		codes.Internal,
		codes.DataLoss:
		return true
	default:
		return false
	}
}

func grpcLevel(code codes.Code) level {
	switch code {
	case codes.Unknown, codes.Internal, codes.DataLoss:
		return levelError
	case codes.DeadlineExceeded,
		codes.PermissionDenied,
		codes.ResourceExhausted,
		codes.FailedPrecondition,
		codes.Aborted,
		codes.Unavailable,
		codes.Unauthenticated:
		return levelWarn
	default:
		return levelInfo
	}
}
