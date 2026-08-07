package middleware

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-kit/kit/endpoint"
	"github.com/stretchr/testify/assert"
)

func TestValidatorMW(t *testing.T) {
	ep := NewCreateUserEndpoint()

	_, err := ep(context.Background(), CreateUserRequest{
		Name:  "A",
		Email: "invalid-email",
		Age:   -5,
	})
	assert.NotNil(t, err)

	_, err = ep(context.Background(), CreateUserRequest{
		Name:  "Alice",
		Email: "alice@example.com",
		Age:   25,
	})
	assert.NoError(t, err)
}

type CreateUserRequest struct {
	Name  string `json:"name" validate:"required,min=2,max=20"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"gte=1,lte=120"`
}

func MakeCreateUserEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CreateUserRequest)
		fmt.Printf("create user: %+v\n", req)
		return map[string]string{"status": "ok"}, nil
	}
}

func NewCreateUserEndpoint() endpoint.Endpoint {
	ep := MakeCreateUserEndpoint()
	ep = ValidatorMW()(ep)
	return ep
}
