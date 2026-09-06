package requestid

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestHTTPMiddlewarePropagatesRequestID(t *testing.T) {
	t.Parallel()

	const want = "request-123"
	var got string
	handler := HTTPMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = FromContext(r.Context())
		if r.Header.Get(Header) != want {
			t.Fatalf("request header = %q, want %q", r.Header.Get(Header), want)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(Header, want)
	handler.ServeHTTP(recorder, request)

	if got != want {
		t.Fatalf("context request ID = %q, want %q", got, want)
	}
	if recorder.Header().Get(Header) != want {
		t.Fatalf("response header = %q, want %q", recorder.Header().Get(Header), want)
	}
}

func TestUnaryClientInterceptorPropagatesRequestID(t *testing.T) {
	t.Parallel()

	ctx := WithContext(context.Background(), "request-123")
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("x-session-key", "session"))
	err := UnaryClientInterceptor()(ctx, "/user.v1.User/GetUser", nil, nil, nil,
		func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
			outgoing, ok := metadata.FromOutgoingContext(ctx)
			if !ok {
				t.Fatal("expected outgoing metadata")
			}
			if got := outgoing.Get(MetadataKey); len(got) != 1 || got[0] != "request-123" {
				t.Fatalf("request ID metadata = %v", got)
			}
			if got := outgoing.Get("x-session-key"); len(got) != 1 || got[0] != "session" {
				t.Fatalf("session metadata = %v", got)
			}
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUnaryServerInterceptorAssignsRequestID(t *testing.T) {
	t.Parallel()

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(MetadataKey, "request-123"))
	_, err := UnaryServerInterceptor()(ctx, nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, _ any) (any, error) {
		if got := FromContext(ctx); got != "request-123" {
			t.Fatalf("context request ID = %q, want request-123", got)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
