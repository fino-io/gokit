package accesslog

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func TestUnaryClientInterceptorPropagatesRequestID(t *testing.T) {
	t.Parallel()

	ctx := withRequestID(context.Background(), "req-client")
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("x-session-key", "session"))
	called := false

	err := UnaryClientInterceptor()(
		ctx,
		"/user.v1.User/GetUser",
		nil,
		nil,
		nil,
		func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
			called = true
			outgoing, ok := metadata.FromOutgoingContext(ctx)
			if !ok {
				t.Fatal("expected outgoing metadata")
			}
			if got := outgoing.Get(requestIDMetadataKey); len(got) != 1 || got[0] != "req-client" {
				t.Fatalf("request ID metadata = %v, want [req-client]", got)
			}
			if got := outgoing.Get("x-session-key"); len(got) != 1 || got[0] != "session" {
				t.Fatalf("session metadata = %v, want [session]", got)
			}
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected interceptor to invoke the RPC")
	}
}

func TestUnaryServerInterceptorLogsStructuredServerError(t *testing.T) {
	t.Parallel()

	logger := &recordingLogger{}
	interceptor := unaryServerInterceptor(config{SampleEvery: 0}, logger.Log)
	ctx := metadata.NewIncomingContext(contextWithTrace(context.Background()), metadata.Pairs("x-request-id", "req-grpc"))
	ctx = peer.NewContext(ctx, &peer.Peer{Addr: &net.TCPAddr{IP: net.ParseIP("203.0.113.11"), Port: 4321}})
	wantErr := status.Error(codes.Unavailable, "unavailable")

	response, err := interceptor(ctx, "request", &grpc.UnaryServerInfo{FullMethod: "/user.v1.User/GetUser"}, func(context.Context, any) (any, error) {
		return "response", wantErr
	})
	if response != "response" || err != wantErr {
		t.Fatalf("response, err = %v, %v; want response, %v", response, err, wantErr)
	}

	entry := logger.single(t)
	if entry.Level != levelWarn {
		t.Fatalf("level = %v, want %v", entry.Level, levelWarn)
	}
	assertField(t, entry.Fields, "protocol", "grpc")
	assertField(t, entry.Fields, "method", "/user.v1.User/GetUser")
	assertField(t, entry.Fields, "code", codes.Unavailable.String())
	assertField(t, entry.Fields, "remote_ip", "203.0.113.11")
	assertField(t, entry.Fields, "request_id", "req-grpc")
	if span := trace.SpanContextFromContext(logger.logContext(t)); span.TraceID() != testTraceID {
		t.Fatalf("trace ID = %s, want %s", span.TraceID(), testTraceID)
	}
}

func TestUnaryServerInterceptorGeneratesRequestIDWhenMissing(t *testing.T) {
	t.Parallel()

	logger := &recordingLogger{}
	interceptor := unaryServerInterceptor(config{SampleEvery: 1}, logger.Log)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/user.v1.User/GetUser"}, func(context.Context, any) (any, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	entry := logger.single(t)
	requestID, ok := entry.Fields["request_id"].(string)
	if !ok {
		t.Fatalf("request ID field = %#v, want string", entry.Fields["request_id"])
	}
	if _, err := uuid.Parse(requestID); err != nil {
		t.Fatalf("request ID = %q, want UUID: %v", requestID, err)
	}
}

func TestUnaryServerInterceptorSkipsSuccessfulHealthMethod(t *testing.T) {
	t.Parallel()

	logger := &recordingLogger{}
	interceptor := unaryServerInterceptor(config{
		SampleEvery: 1,
		GRPC:        grpcConfig{SkipMethods: []string{"/grpc.health.v1.Health/Check"}},
	}, logger.Log)

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/grpc.health.v1.Health/Check"}, func(context.Context, any) (any, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := logger.count(); got != 0 {
		t.Fatalf("log count = %d, want 0", got)
	}
}

func TestUnaryServerInterceptorLogsSlowSuccessfulCall(t *testing.T) {
	t.Parallel()

	logger := &recordingLogger{}
	interceptor := unaryServerInterceptor(config{SlowThreshold: time.Nanosecond}, logger.Log)
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/user.v1.User/ListUsers"}, func(context.Context, any) (any, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if entry := logger.single(t); entry.Level != levelInfo {
		t.Fatalf("level = %v, want %v", entry.Level, levelInfo)
	}
}

func TestUnaryServerInterceptorLogsAndPropagatesPanic(t *testing.T) {
	t.Parallel()

	logger := &recordingLogger{}
	interceptor := unaryServerInterceptor(config{SampleEvery: 0}, logger.Log)

	defer func() {
		if recovered := recover(); recovered != "boom" {
			t.Fatalf("recovered = %v, want boom", recovered)
		}
		if entry := logger.single(t); entry.Level != levelError {
			t.Fatalf("level = %v, want %v", entry.Level, levelError)
		}
	}()
	_, _ = interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/user.v1.User/GetUser"}, func(context.Context, any) (any, error) {
		panic("boom")
	})
}
