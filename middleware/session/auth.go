package session

import (
	"context"

	gokitsession "github.com/fino-io/gokit/session"
)

func contextWithValidatedSession(ctx context.Context, validator gokitsession.Validator, token string) (context.Context, error) {
	session, err := validator.Validate(ctx, token)
	if err != nil {
		return ctx, err
	}

	ctx = gokitsession.WithToken(ctx, token)
	ctx = gokitsession.WithSession(ctx, session)
	return ctx, nil
}
