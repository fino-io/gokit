package session

import (
	"context"
	"fmt"

	gokitsession "github.com/fino-io/gokit/session"
	"github.com/go-kit/kit/endpoint"
)

func ValidateMiddleware(validator gokitsession.Validator) endpoint.Middleware {
	if validator == nil {
		return errorMiddleware(gokitsession.ErrValidatorRequired)
	}

	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			token, ok := gokitsession.TokenFromContext(ctx)
			if !ok {
				return nil, gokitsession.ErrTokenRequired
			}

			ctx, err := contextWithValidatedSession(ctx, validator, token)
			if err != nil {
				return nil, fmt.Errorf("validate session: %w", err)
			}

			return next(ctx, req)
		}
	}
}

func AuthenticateMiddleware(validator gokitsession.Validator, resolver UserResolver) endpoint.Middleware {
	if resolver == nil {
		return errorMiddleware(ErrUserResolverRequired)
	}

	validate := ValidateMiddleware(validator)

	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return validate(func(ctx context.Context, req any) (any, error) {
			session, ok := gokitsession.SessionFromContext(ctx)
			if !ok {
				return nil, gokitsession.ErrSessionNotFound
			}

			resolved, err := resolver.ResolveUser(ctx, session, req)
			if err != nil {
				return nil, fmt.Errorf("resolve user: %w", err)
			}
			if resolved == nil || resolved.ID == "" || resolved.Value == nil {
				return nil, ErrResolvedUserInvalid
			}
			if resolved.ID != session.Subject.UserID {
				return nil, ErrResolvedUserMismatch
			}

			return next(WithUser(ctx, resolved.Value), req)
		})
	}
}

func errorMiddleware(err error) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			return nil, err
		}
	}
}
