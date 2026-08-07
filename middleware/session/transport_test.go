package session

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gokitsession "github.com/fino-io/gokit/session"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestHTTPMiddlewareReadsHeaderAndWritesContext(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	manager := newTestManager(t, gokitsession.NewMemoryStore(), testClock(&now), testIDSequence("session-1"))

	issued, err := manager.Issue(context.Background(), gokitsession.Subject{UserID: "user-1"})
	require.NoError(t, err)

	handler := HTTPMiddleware(manager)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := gokitsession.TokenFromContext(r.Context())
		require.True(t, ok)
		require.Equal(t, issued.Token, token)

		session, ok := gokitsession.SessionFromContext(r.Context())
		require.True(t, ok)
		require.Equal(t, issued.Session, session)

		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(AuthorizationHeader, "Bearer "+issued.Token)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestHTTPMiddlewareReadsCookie(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	manager := newTestManager(t, gokitsession.NewMemoryStore(), testClock(&now), testIDSequence("session-1"))

	issued, err := manager.Issue(context.Background(), gokitsession.Subject{UserID: "user-1"})
	require.NoError(t, err)

	handler := HTTPMiddleware(manager)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := gokitsession.SessionFromContext(r.Context())
		require.True(t, ok)
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: SessionTokenCookie, Value: issued.Token})
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestHTTPMiddlewareUsesCustomTokenHeader(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	manager := newTestManager(t, gokitsession.NewMemoryStore(), testClock(&now), testIDSequence("session-1"))

	issued, err := manager.Issue(context.Background(), gokitsession.Subject{UserID: "user-1"})
	require.NoError(t, err)

	handler := HTTPMiddleware(manager, WithTokenHeader("X-Auth-Token"))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := gokitsession.SessionFromContext(r.Context())
		require.True(t, ok)
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Auth-Token", issued.Token)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestUnaryServerInterceptorReadsMetadata(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	manager := newTestManager(t, gokitsession.NewMemoryStore(), testClock(&now), testIDSequence("session-1"))

	issued, err := manager.Issue(context.Background(), gokitsession.Subject{UserID: "user-1"})
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+issued.Token))
	interceptor := UnaryServerInterceptor(manager)

	resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, req any) (any, error) {
		session, ok := gokitsession.SessionFromContext(ctx)
		require.True(t, ok)
		require.Equal(t, issued.Session, session)
		return "ok", nil
	})

	require.NoError(t, err)
	require.Equal(t, "ok", resp)
}

func TestStreamServerInterceptorReadsMetadata(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	manager := newTestManager(t, gokitsession.NewMemoryStore(), testClock(&now), testIDSequence("session-1"))

	issued, err := manager.Issue(context.Background(), gokitsession.Subject{UserID: "user-1"})
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-session-token", issued.Token))
	interceptor := StreamServerInterceptor(manager)

	err = interceptor(nil, mockServerStream{ctx: ctx}, &grpc.StreamServerInfo{}, func(srv any, stream grpc.ServerStream) error {
		session, ok := gokitsession.SessionFromContext(stream.Context())
		require.True(t, ok)
		require.Equal(t, issued.Session, session)
		return nil
	})

	require.NoError(t, err)
}

func TestUnaryServerInterceptorUsesCustomBearerHeader(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	manager := newTestManager(t, gokitsession.NewMemoryStore(), testClock(&now), testIDSequence("session-1"))

	issued, err := manager.Issue(context.Background(), gokitsession.Subject{UserID: "user-1"})
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-auth", "Bearer "+issued.Token))
	interceptor := UnaryServerInterceptor(manager, WithBearerHeader("X-Auth"))

	resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, req any) (any, error) {
		_, ok := gokitsession.SessionFromContext(ctx)
		require.True(t, ok)
		return "ok", nil
	})

	require.NoError(t, err)
	require.Equal(t, "ok", resp)
}

func TestUnaryServerInterceptorRejectsMissingToken(t *testing.T) {
	interceptor := UnaryServerInterceptor(noopValidator{})

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, req any) (any, error) {
		t.Fatal("handler should not be called")
		return nil, nil
	})

	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

type noopValidator struct{}

func (noopValidator) Validate(ctx context.Context, token string) (*gokitsession.Session, error) {
	return &gokitsession.Session{ID: "session-1", Subject: gokitsession.Subject{UserID: "user-1"}}, nil
}

type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s mockServerStream) Context() context.Context {
	return s.ctx
}
