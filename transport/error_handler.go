package transport

import (
	"context"
	"errors"
	"net/http"

	"github.com/fino-io/finokit/logs"
	nhttp "github.com/fino-io/gokit/transport/http"
	kittransport "github.com/go-kit/kit/transport"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorHandler logs unexpected server errors with the request context.
func ErrorHandler() kittransport.ErrorHandler {
	return kittransport.ErrorHandlerFunc(func(ctx context.Context, err error) {
		if !shouldLogError(err) {
			return
		}
		logs.Ctx(ctx).Errorw("transport request failed", "error", err)
	})
}

func shouldLogError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if code := nhttp.StatusCodeFromError(err); code >= http.StatusBadRequest && code < http.StatusInternalServerError {
		return false
	}

	switch status.Code(err) {
	case codes.Canceled,
		codes.DeadlineExceeded,
		codes.InvalidArgument,
		codes.NotFound,
		codes.AlreadyExists,
		codes.Aborted,
		codes.PermissionDenied,
		codes.ResourceExhausted,
		codes.FailedPrecondition,
		codes.Unauthenticated,
		codes.OutOfRange:
		return false
	default:
		return true
	}
}
