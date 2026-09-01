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

// UnaryServerInterceptor logs completed unary gRPC server calls.
func UnaryServerInterceptor(cfg Config) grpc.UnaryServerInterceptor {
	return unaryServerInterceptor(cfg, defaultLog)
}

func unaryServerInterceptor(cfg Config, log logFunc) grpc.UnaryServerInterceptor {
	policy := newPolicy(cfg, cfg.GRPC.SkipMethods)
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (response any, err error) {
		started := time.Now()
		defer func() {
			recovered := recover()
			code := status.Code(err)
			if recovered != nil {
				code = codes.Internal
			}
			logGRPC(ctx, info.FullMethod, code, time.Since(started), policy, log)
			if recovered != nil {
				panic(recovered)
			}
		}()
		return handler(ctx, req)
	}
}

func logGRPC(ctx context.Context, method string, code codes.Code, duration time.Duration, policy *policy, log logFunc) {
	requestID := requestIDFromMetadata(ctx)
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

func requestIDFromMetadata(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get("x-request-id")
	if len(values) == 0 {
		return ""
	}
	return values[0]
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
