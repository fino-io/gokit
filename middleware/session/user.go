package session

import (
	"context"
	"errors"

	gokitsession "github.com/fino-io/gokit/session"
)

var (
	ErrUserResolverRequired = errors.New("session user resolver is required")
	ErrResolvedUserInvalid  = errors.New("resolved user is invalid")
	ErrResolvedUserMismatch = errors.New("resolved user does not match session subject")
)

type ResolvedUser struct {
	ID    string
	Value any
}

type UserResolver interface {
	ResolveUser(ctx context.Context, session *gokitsession.Session, req any) (*ResolvedUser, error)
}

type UserResolverFunc func(ctx context.Context, session *gokitsession.Session, req any) (*ResolvedUser, error)

func (f UserResolverFunc) ResolveUser(ctx context.Context, session *gokitsession.Session, req any) (*ResolvedUser, error) {
	return f(ctx, session, req)
}
