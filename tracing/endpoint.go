package tracing

import (
	"context"
	"errors"
	"strconv"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/sd/lb"
)

// TraceEndpoint returns a Middleware that wraps the `next` Endpoint in an
// OpenTelemetry Span called `operationName`.
//
// If `ctx` already has a Span, child span is created from it.
// If `ctx` doesn't yet have a Span, the new one is created.
func TraceEndpoint(tracer trace.Tracer, operationName string, opts ...EndpointOption) endpoint.Middleware {
	cfg := &EndpointOptions{
		Attributes: make([]attribute.KeyValue, 0),
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			spanName := operationName
			if cfg.GetOperationName != nil {
				if newOperationName := cfg.GetOperationName(ctx, operationName); newOperationName != "" {
					spanName = newOperationName
				}
			}

			ctx, span := tracer.Start(ctx, spanName, cfg.SpanStartOptions...)
			defer span.End()

			applyAttributes(span, cfg.Attributes)
			if cfg.GetAttributes != nil {
				extraAttributes := cfg.GetAttributes(ctx)
				applyAttributes(span, extraAttributes)
			}

			defer func() {
				if err != nil {
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
					var lbErr lb.RetryError
					if errors.As(err, &lbErr) {
						// handle errors originating from lb.Retry
						for idx, rawErr := range lbErr.RawErrors {
							if rawErr != nil {
								span.SetAttributes(attribute.String("gokit.retry.error."+strconv.Itoa(idx+1), rawErr.Error()))
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
					span.RecordError(businessErr)
					span.SetAttributes(attribute.String("gokit.business.error", businessErr.Error()))

					if cfg.IgnoreBusinessError {
						return
					}

					// treating business error as real error in span.
					span.SetStatus(codes.Error, businessErr.Error())

					return
				}
			}()

			return next(ctx, request)
		}
	}
}

// TraceClient returns a Middleware that wraps the `next` Endpoint in an
// OpenTelemetry Span called `operationName` with client span kind.
func TraceClient(tracer trace.Tracer, operationName string, opts ...EndpointOption) endpoint.Middleware {
	opts = append(opts, WithSpanStartOptions(trace.WithSpanKind(trace.SpanKindClient)))

	return TraceEndpoint(tracer, operationName, opts...)
}

func applyAttributes(span trace.Span, attributes []attribute.KeyValue) {
	span.SetAttributes(attributes...)
}
