package middleware

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
	"golang.org/x/time/rate"

	"github.com/fino-io/core/go/fino/core"
)

func NewTokenBucketLimitMW(bkt *rate.Limiter) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			if !bkt.Allow() {
				return nil, core.NewResourceExhaustedError("Rate limit exceed!")
			}
			return next(ctx, request)
		}
	}
}

func EveryRateLimiter(interval time.Duration, b int) endpoint.Middleware {
	limiter := rate.NewLimiter(rate.Every(interval), b)
	return NewTokenBucketLimitMW(limiter)
}
