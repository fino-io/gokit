package tracing

import (
	"context"
	"errors"
	"strconv"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/sd/lb"
)

// TraceEndpoint returns a Middleware that wraps the `next` Endpoint in an
// OpenTelemetry Span called `operationName`.
//
// If `ctx` already has a Span, child span is created from it.
// If `ctx` doesn't yet have a Span, the new one is created.
func TraceEndpoint(tracer trace.Tracer, operationName string) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			ctx, span := tracer.Start(ctx, operationName)
			defer span.End()

			defer func() {
				if err != nil {
					recordError(span, err)
					var lbErr lb.RetryError
					if errors.As(err, &lbErr) {
						// handle errors originating from lb.Retry
						for idx, rawErr := range lbErr.RawErrors {
							if rawErr != nil {
								span.SetAttributes(attribute.String("gokit.retry.error."+strconv.Itoa(idx+1), telemetryString(rawErr.Error())))
							}
						}

						return
					}

					// generic error
					return
				}

				// test for business error
				if res, ok := response.(endpoint.Failer); ok && res.Failed() != nil {
					businessErr := res.Failed()
					recordError(span, businessErr)
					span.SetAttributes(attribute.String("gokit.business.error", telemetryString(businessErr.Error())))
					return
				}
			}()

			return next(ctx, request)
		}
	}
}
