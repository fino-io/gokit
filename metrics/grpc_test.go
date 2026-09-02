package metrics

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGRPCServerRecordsCode(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewGRPCServer(registry, "test")

	_, _ = metrics.UnaryInterceptor()(
		context.Background(), nil,
		&grpc.UnaryServerInfo{FullMethod: "/mailgate.Service/CreateTask"},
		func(context.Context, any) (any, error) {
			return nil, status.Error(codes.InvalidArgument, "invalid")
		},
	)

	if got := counterValue(t, registry, "test_grpc_server_requests_total", map[string]string{
		"service": "mailgate.Service", "method": "CreateTask", "code": "InvalidArgument",
	}); got != 1 {
		t.Fatalf("requests = %v, want 1", got)
	}
}
