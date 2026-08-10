package lb

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
)

// RetryError is an error wrapper that is used by the retry mechanism. All
// errors returned by the retry mechanism via its endpoint will be RetryErrors.
type RetryError struct {
	Final    error
	Attempts int
}

func (e RetryError) Unwrap() error { return e.Final }

func (e RetryError) Error() string { return e.Final.Error() }

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
// timeout is elapsed, whichever comes first. Interval controls the delay
// between attempts; zero retries immediately.
func Retry(max int, timeout, interval time.Duration, b Balancer) endpoint.Endpoint {
	return RetryWithCallback(timeout, interval, b, maxRetries(max))
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
// first. Interval controls the delay between attempts; zero retries immediately.
func RetryWithCallback(timeout, interval time.Duration, b Balancer, cb Callback) endpoint.Endpoint {
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
				final.Attempts = i - 1
				return nil, final
			}

			e, err := b.Endpoint()
			if err == nil {
				response, err = e(newctx, request)
			}
			if err != nil {
				if newctx.Err() != nil {
					final.Final = newctx.Err()
					final.Attempts = i
					return nil, final
				}

				keepTrying, replacement := cb(i, err)
				if replacement != nil {
					err = replacement
				}
				if !keepTrying {
					final.Final = err
					final.Attempts = i
					return nil, final
				}

				if !wait(newctx, interval) {
					final.Final = newctx.Err()
					final.Attempts = i
					return nil, final
				}
				continue
			}

			return response, nil
		}
	}
}

func wait(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
