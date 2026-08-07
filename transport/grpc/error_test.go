package grpc

import (
	"errors"
	"testing"

	"github.com/fino-io/core/go/fino/core"
	"google.golang.org/grpc/codes"
)

func TestCodeFromCoreError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"bad request", core.NewBadRequestError("bad request"), codes.InvalidArgument},
		{"invalid argument", core.NewInvalidArgumentError("invalid argument"), codes.InvalidArgument},
		{"malformed request", core.NewMalformedRequestError("malformed request"), codes.InvalidArgument},
		{"failed precondition", core.NewFailedPreconditionError("failed precondition"), codes.FailedPrecondition},
		{"out of range", core.NewOutOfRangeError("out of range"), codes.OutOfRange},
		{"unauthenticated", core.NewUnauthenticatedError("unauthenticated"), codes.Unauthenticated},
		{"permission denied", core.NewPermissionDeniedError("permission denied"), codes.PermissionDenied},
		{"not found", core.NewNotFoundError("not found"), codes.NotFound},
		{"already exists", core.NewAlreadyExistsError("already exists"), codes.AlreadyExists},
		{"aborted", core.NewAbortedError("aborted"), codes.Aborted},
		{"resource exhausted", core.NewResourceExhaustedError("resource exhausted"), codes.ResourceExhausted},
		{"cancelled", core.NewCancelledError("cancelled"), codes.Canceled},
		{"unknown", core.NewUnknownErrorError("unknown"), codes.Unknown},
		{"internal", core.NewInternalErrorError("internal"), codes.Internal},
		{"data loss", core.NewDataLossError("data loss"), codes.DataLoss},
		{"unimplemented", core.NewUnimplementedError("unimplemented"), codes.Unimplemented},
		{"unavailable", core.NewUnavailableError("unavailable"), codes.Unavailable},
		{"deadline exceeded", core.NewDeadlineExceededError("deadline exceeded"), codes.DeadlineExceeded},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			code, ok := CodeFromCoreError(test.err)
			if !ok || code != test.code {
				t.Fatalf("CodeFromCoreError() = (%s, %t), want (%s, true)", code, ok, test.code)
			}
		})
	}

	if _, ok := CodeFromCoreError(errors.New("unknown")); ok {
		t.Fatal("expected an ordinary error to be unmapped")
	}
}
