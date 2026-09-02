package transport

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestShouldLogError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "canceled", err: context.Canceled, want: false},
		{name: "deadline exceeded", err: context.DeadlineExceeded, want: false},
		{name: "http client error", err: statusCoderError(http.StatusUnauthorized), want: false},
		{name: "wrapped http client error", err: fmt.Errorf("wrapped: %w", statusCoderError(http.StatusNotFound)), want: false},
		{name: "grpc client error", err: status.Error(codes.Unauthenticated, "authentication required"), want: false},
		{name: "grpc deadline exceeded", err: status.Error(codes.DeadlineExceeded, "deadline exceeded"), want: false},
		{name: "http server error", err: statusCoderError(http.StatusInternalServerError), want: true},
		{name: "grpc server error", err: status.Error(codes.Unavailable, "service unavailable"), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, shouldLogError(tt.err))
		})
	}
}

type statusCoderError int

func (e statusCoderError) Error() string {
	return http.StatusText(int(e))
}

func (e statusCoderError) StatusCode() int {
	return int(e)
}
