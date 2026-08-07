package session

import (
	"context"
	"errors"
	"testing"
	"time"

	gokitsession "github.com/fino-io/gokit/session"
	"github.com/stretchr/testify/require"
)

func TestValidateMiddleware(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	manager := newTestManager(t, gokitsession.NewMemoryStore(), testClock(&now), testIDSequence("session-1"))

	issued, err := manager.Issue(context.Background(), gokitsession.Subject{UserID: "user-1"})
	require.NoError(t, err)

	endpoint := ValidateMiddleware(manager)(func(ctx context.Context, req any) (any, error) {
		session, ok := gokitsession.SessionFromContext(ctx)
		require.True(t, ok)
		require.Equal(t, issued.Session, session)
		return "ok", nil
	})

	resp, err := endpoint(gokitsession.WithToken(context.Background(), issued.Token), nil)
	require.NoError(t, err)
	require.Equal(t, "ok", resp)
}

func TestAuthenticateMiddleware(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	manager := newTestManager(t, gokitsession.NewMemoryStore(), testClock(&now), testIDSequence("session-1"))

	issued, err := manager.Issue(context.Background(), gokitsession.Subject{UserID: "user-1"})
	require.NoError(t, err)

	resolver := UserResolverFunc(func(ctx context.Context, session *gokitsession.Session, req any) (*ResolvedUser, error) {
		return &ResolvedUser{
			ID:    "user-1",
			Value: &testUser{ID: "user-1", Name: "John Doe"},
		}, nil
	})

	endpoint := AuthenticateMiddleware(manager, resolver)(func(ctx context.Context, req any) (any, error) {
		user, ok := UserFromContext[*testUser](ctx)
		require.True(t, ok)
		require.Equal(t, "user-1", user.ID)
		return "ok", nil
	})

	resp, err := endpoint(gokitsession.WithToken(context.Background(), issued.Token), nil)
	require.NoError(t, err)
	require.Equal(t, "ok", resp)
}

func TestAuthenticateMiddlewareErrors(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	manager := newTestManager(t, gokitsession.NewMemoryStore(), testClock(&now), testIDSequence("session-1"))

	issued, err := manager.Issue(context.Background(), gokitsession.Subject{UserID: "user-1"})
	require.NoError(t, err)

	t.Run("validator required", func(t *testing.T) {
		endpoint := ValidateMiddleware(nil)(func(ctx context.Context, req any) (any, error) {
			t.Fatal("next should not be called")
			return nil, nil
		})

		_, err := endpoint(context.Background(), nil)
		require.ErrorIs(t, err, gokitsession.ErrValidatorRequired)
	})

	t.Run("resolver required", func(t *testing.T) {
		endpoint := AuthenticateMiddleware(manager, nil)(func(ctx context.Context, req any) (any, error) {
			t.Fatal("next should not be called")
			return nil, nil
		})

		_, err := endpoint(context.Background(), nil)
		require.ErrorIs(t, err, ErrUserResolverRequired)
	})

	t.Run("resolved user mismatch", func(t *testing.T) {
		endpoint := AuthenticateMiddleware(manager, UserResolverFunc(func(ctx context.Context, session *gokitsession.Session, req any) (*ResolvedUser, error) {
			return &ResolvedUser{ID: "other-user", Value: &testUser{ID: "other-user"}}, nil
		}))(func(ctx context.Context, req any) (any, error) {
			t.Fatal("next should not be called")
			return nil, nil
		})

		_, err := endpoint(gokitsession.WithToken(context.Background(), issued.Token), nil)
		require.ErrorIs(t, err, ErrResolvedUserMismatch)
	})

	t.Run("resolver error", func(t *testing.T) {
		endpoint := AuthenticateMiddleware(manager, UserResolverFunc(func(ctx context.Context, session *gokitsession.Session, req any) (*ResolvedUser, error) {
			return nil, errors.New("db down")
		}))(func(ctx context.Context, req any) (any, error) {
			t.Fatal("next should not be called")
			return nil, nil
		})

		_, err := endpoint(gokitsession.WithToken(context.Background(), issued.Token), nil)
		require.ErrorContains(t, err, "resolve user")
	})
}
