package lb

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-kit/kit/endpoint"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RetryError is an error wrapper that is used by the retry mechanism. All
// errors returned by the retry mechanism via its endpoint will be RetryErrors.
type RetryError struct {
	RawErrors []error // all errors encountered from endpoints directly
	Final     error   // the final, terminating error
}

func (e RetryError) Unwrap() error { return e.Final }

func (e RetryError) Error() string {
	var suffix string
	n := len(e.RawErrors)
	if n > 0 && errors.Is(e.RawErrors[n-1], e.Final) {
		n--
	}
	if n > 0 {
		a := make([]string, n)
		for i := 0; i < n; i++ {
			a[i] = e.RawErrors[i].Error()
		}
		suffix = fmt.Sprintf(" (previously: %s)", strings.Join(a, "; "))
	}
	return fmt.Sprintf("%v%s", e.Final, suffix)
}

// Callback is a function that is given the current attempt count and the error
// received from the underlying endpoint. It should return whether the Retry
// function should continue trying to get a working endpoint, and a custom error
// if desired. The error message may be nil, but a true/false is always
// expected. In all cases, if the replacement error is supplied, the received
// error will be replaced in the calling context.
type Callback func(n int, received error) (keepTrying bool, replacement error)

// Retry wraps a service load balancer and returns an endpoint oriented load
// balancer for the specified service method. Requests to the endpoint will be
// automatically load balanced via the load balancer. Requests that return
// errors will be retried until they succeed, up to max times, or until the
// timeout is elapsed, whichever comes first.
func Retry(max int, timeout time.Duration, b Balancer) endpoint.Endpoint {
	return RetryWithCallback(timeout, b, maxRetries(max))
}

func maxRetries(max int) Callback {
	return func(n int, err error) (keepTrying bool, replacement error) {
		return n < max, nil
	}
}

func alwaysRetry(int, error) (keepTrying bool, replacement error) {
	return true, nil
}

// RetryWithCallback wraps a service load balancer and returns an endpoint
// oriented load balancer for the specified service method. Requests to the
// endpoint will be automatically load balanced via the load balancer. Requests
// that return errors will be retried until they succeed, up to max times, until
// the callback returns false, or until the timeout is elapsed, whichever comes
// first.
func RetryWithCallback(timeout time.Duration, b Balancer, cb Callback) endpoint.Endpoint {
	if cb == nil {
		cb = alwaysRetry
	}

	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		if b == nil {
			return nil, ErrNilBalancer
		}

		var (
			newctx = ctx
			cancel context.CancelFunc
			final  RetryError
		)
		if timeout > 0 {
			newctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		for i := 1; ; i++ {
			if err := newctx.Err(); err != nil {
				final.Final = err
				return nil, final
			}

			e, err := b.Endpoint()
			if err == nil {
				response, err = invoke(newctx, e, request)
			}
			if err != nil {
				if newctx.Err() != nil {
					final.Final = newctx.Err()
					return nil, final
				}
				final.RawErrors = append(final.RawErrors, err)

				/*
					// use https://github.com/grpc/grpc/blob/master/doc/http-grpc-status-mapping.md
					// https://github.com/grpc/grpc/blob/master/doc/statuscodes.md
					// for mapping
					func (e *Error) GRPCStatus() *status.Status {
						var code codes.Code
						switch e.Code.Value {
						case http.StatusOK:
							code = codes.OK
						case http.StatusInternalServerError:
							code = codes.Internal
						case http.StatusBadRequest:
							code = codes.InvalidArgument
						case http.StatusGatewayTimeout:
							code = codes.DeadlineExceeded
						case http.StatusNotFound:
							code = codes.NotFound
						case http.StatusConflict:
							code = codes.AlreadyExists
						case http.StatusForbidden:
							code = codes.PermissionDenied
						case http.StatusUnauthorized:
							code = codes.Unauthenticated
						case http.StatusNotImplemented:
							code = codes.Unimplemented
						case http.StatusServiceUnavailable:
							code = codes.Unavailable
						case http.StatusTooManyRequests:
							code = codes.ResourceExhausted
						default:
							code = codes.Unknown
						}
						return status.New(code, e.Message)
					}
				*/
				if v, ok := status.FromError(err); ok {
					code := v.Code()
					if code == codes.InvalidArgument || code == codes.NotFound ||
						code == codes.Unimplemented || code == codes.Unauthenticated {
						// app not found error is not error
						final.Final = err
						return nil, final
					}
				}

				keepTrying, replacement := cb(i, err)
				if replacement != nil {
					err = replacement
				}
				if !keepTrying {
					final.Final = err
					return nil, final
				}

				if !wait(newctx, time.Millisecond) {
					final.Final = newctx.Err()
					return nil, final
				}
				continue
			}

			return response, nil
		}
	}
}

func wait(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func invoke(ctx context.Context, e endpoint.Endpoint, request interface{}) (interface{}, error) {
	type result struct {
		response interface{}
		err      error
	}

	done := make(chan result, 1)
	go func() {
		response, err := e(ctx, request)
		done <- result{response: response, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-done:
		return r.response, r.err
	}
}
