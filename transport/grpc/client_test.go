package grpc

import (
	"context"
	"testing"

	"github.com/fino-io/gokit/metrics"
	"go.opentelemetry.io/otel/trace"
	stdgrpc "google.golang.org/grpc"
)

func TestWithClientObservabilityAddsTransparentUnaryInterceptors(t *testing.T) {
	var options clientOptions
	WithClientObservability(
		"caller-service",
		"user.v1.UserService",
		trace.NewNoopTracerProvider().Tracer("test"),
		metrics.Disabled(),
	)(&options)

	if len(options.unary) != 3 {
		t.Fatalf("unary interceptor count = %d, want 3", len(options.unary))
	}

	for _, interceptor := range options.unary {
		called := false
		err := interceptor(
			context.Background(),
			"/user.v1.UserService/GetUser",
			nil,
			nil,
			nil,
			func(context.Context, string, any, any, *stdgrpc.ClientConn, ...stdgrpc.CallOption) error {
				called = true
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
}

func TestNewClientRequiresTransportCredentials(t *testing.T) {
	conn, err := NewClient("user.v1.UserService")
	if err == nil {
		_ = conn.Close()
		t.Fatal("expected missing transport credentials error")
	}
}

func TestNewClientAcceptsExplicitInsecureTransport(t *testing.T) {
	conn, err := NewClient("user.v1.UserService", WithInsecure())
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("close client: %v", err)
	}
}
