package metrics

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGRPCClientRecordsCalls(t *testing.T) {
	tests := []struct {
		name string
		ctx  func() context.Context
		err  error
		code codes.Code
	}{
		{name: "success", ctx: context.Background, code: codes.OK},
		{name: "grpc error", ctx: context.Background, err: status.Error(codes.InvalidArgument, "invalid"), code: codes.InvalidArgument},
		{name: "deadline exceeded", ctx: context.Background, err: context.DeadlineExceeded, code: codes.DeadlineExceeded},
		{name: "canceled", ctx: context.Background, err: context.Canceled, code: codes.Canceled},
		{name: "non grpc error", ctx: context.Background, err: errors.New("failed"), code: codes.Unknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := prometheus.NewRegistry()
			metrics := NewGRPCClient(registry, "test", "mailgate-client", "mailgate.v1.MailgateService")
			interceptor := metrics.UnaryInterceptor()

			err := interceptor(
				tt.ctx(),
				"/mailgate.v1.MailgateService/CreateTask",
				nil,
				nil,
				nil,
				func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
					return tt.err
				},
			)
			if !errors.Is(err, tt.err) {
				t.Fatalf("error = %v, want %v", err, tt.err)
			}

			labels := map[string]string{
				"client":  "mailgate-client",
				"target":  "mailgate.v1.MailgateService",
				"service": "mailgate.v1.MailgateService",
				"method":  "CreateTask",
				"code":    tt.code.String(),
			}
			if got := counterValue(t, registry, "test_grpc_client_requests_total", labels); got != 1 {
				t.Fatalf("requests = %v, want 1", got)
			}
			if got := gaugeValue(t, registry, "test_grpc_client_requests_in_flight", labelsWithout(labels, "code")); got != 0 {
				t.Fatalf("in flight = %v, want 0", got)
			}
			if got := histogramCount(t, registry, "test_grpc_client_request_duration_seconds", labelsWithout(labels, "code")); got != 1 {
				t.Fatalf("duration count = %v, want 1", got)
			}
		})
	}
}

func TestGRPCClientTracksInFlightCall(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewGRPCClient(registry, "test", "mailgate-client", "mailgate.v1.MailgateService")
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)

	go func() {
		done <- metrics.UnaryInterceptor()(
			context.Background(),
			"/mailgate.v1.MailgateService/GetTask",
			nil,
			nil,
			nil,
			func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
				close(started)
				<-release
				return nil
			},
		)
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("invoker did not start")
	}
	labels := map[string]string{
		"client": "mailgate-client", "target": "mailgate.v1.MailgateService",
		"service": "mailgate.v1.MailgateService", "method": "GetTask",
	}
	if got := gaugeValue(t, registry, "test_grpc_client_requests_in_flight", labels); got != 1 {
		t.Fatalf("in flight = %v, want 1", got)
	}

	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if got := gaugeValue(t, registry, "test_grpc_client_requests_in_flight", labels); got != 0 {
		t.Fatalf("in flight after call = %v, want 0", got)
	}
}

func TestSplitMethodRejectsUnstableValues(t *testing.T) {
	tests := []struct {
		full    string
		service string
		method  string
	}{
		{full: "/mailgate.v1.MailgateService/CreateTask", service: "mailgate.v1.MailgateService", method: "CreateTask"},
		{full: "invalid", service: "unknown", method: "unknown"},
		{full: "/too/many/parts", service: "unknown", method: "unknown"},
	}
	for _, tt := range tests {
		service, method := splitMethod(tt.full)
		if service != tt.service || method != tt.method {
			t.Fatalf("splitMethod(%q) = %q, %q; want %q, %q", tt.full, service, method, tt.service, tt.method)
		}
	}
}

func labelsWithout(labels map[string]string, key string) map[string]string {
	result := make(map[string]string, len(labels)-1)
	for name, value := range labels {
		if name != key {
			result[name] = value
		}
	}
	return result
}
