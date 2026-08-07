package grpc

import (
	"github.com/fino-io/core/go/fino/core"
	"google.golang.org/grpc/codes"
)

// CodeFromCoreError maps a core error category to its gRPC equivalent.
func CodeFromCoreError(err error) (codes.Code, bool) {
	switch {
	case core.IsBadRequestError(err), core.IsInvalidArgumentError(err), core.IsMalformedRequestError(err):
		return codes.InvalidArgument, true
	case core.IsFailedPreconditionError(err):
		return codes.FailedPrecondition, true
	case core.IsOutOfRangeError(err):
		return codes.OutOfRange, true
	case core.IsUnauthenticatedError(err):
		return codes.Unauthenticated, true
	case core.IsPermissionDeniedError(err):
		return codes.PermissionDenied, true
	case core.IsNotFoundError(err):
		return codes.NotFound, true
	case core.IsAlreadyExistsError(err):
		return codes.AlreadyExists, true
	case core.IsAbortedError(err):
		return codes.Aborted, true
	case core.IsResourceExhaustedError(err):
		return codes.ResourceExhausted, true
	case core.IsCancelledError(err):
		return codes.Canceled, true
	case core.IsUnknownErrorError(err):
		return codes.Unknown, true
	case core.IsInternalError(err):
		return codes.Internal, true
	case core.IsDataLossError(err):
		return codes.DataLoss, true
	case core.IsUnimplementedError(err):
		return codes.Unimplemented, true
	case core.IsUnavailableError(err):
		return codes.Unavailable, true
	case core.IsDeadlineExceededError(err):
		return codes.DeadlineExceeded, true
	default:
		return codes.Unknown, false
	}
}
