package tracing

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	traceSDK "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

func newTracer(ctx context.Context, serviceName string, cfg *config) (trace.Tracer, ShutdownFunc, error) {
	if cfg == nil {
		return noopTracer(serviceName), noopShutdown, errors.New("tracing config is nil")
	}
	if !cfg.Enable {
		return noopTracer(serviceName), noopShutdown, nil
	}

	provider, err := newTraceProvider(ctx, serviceName, cfg)
	if err != nil {
		return noopTracer(serviceName), noopShutdown, err
	}

	otel.SetTracerProvider(provider)
	tracer := provider.Tracer(serviceName)
	return tracer, provider.Shutdown, nil
}

func newTraceProvider(ctx context.Context, serviceName string, cfg *config) (*traceSDK.TracerProvider, error) {
	if cfg == nil {
		cfg = &config{SampleRatio: 1}
	}
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = defaultOTLPEndpoint
	}

	options := []otlptracehttp.Option{otlptracehttp.WithInsecure()}
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		options = append(options, otlptracehttp.WithEndpointURL(endpoint))
	} else {
		options = append(options, otlptracehttp.WithEndpoint(endpoint))
	}

	exp, err := otlptracehttp.New(ctx, options...)
	if err != nil {
		return nil, err
	}

	res, err := newResource(ctx, serviceName, cfg)
	if err != nil {
		return nil, err
	}

	traceProvider := traceSDK.NewTracerProvider(
		traceSDK.WithResource(res),
		traceSDK.WithSampler(samplerFromRatio(sampleRatio(cfg))),
		traceSDK.WithBatcher(exp, traceSDK.WithBatchTimeout(time.Second)),
	)

	return traceProvider, nil
}

func sampleRatio(cfg *config) float64 {
	if cfg == nil || cfg.SampleRatio == 0 {
		return 1
	}
	return cfg.SampleRatio
}

func samplerFromRatio(ratio float64) traceSDK.Sampler {
	if ratio <= 0 {
		return traceSDK.ParentBased(traceSDK.NeverSample())
	}
	if ratio >= 1 {
		return traceSDK.ParentBased(traceSDK.AlwaysSample())
	}
	return traceSDK.ParentBased(traceSDK.TraceIDRatioBased(ratio))
}

func newResource(ctx context.Context, serviceName string, cfg *config) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{
		semconv.ServiceName(serviceName),
	}
	if cfg != nil {
		if cfg.Environment != "" {
			attrs = append(attrs, attribute.String("deployment.environment", cfg.Environment))
		}
		if cfg.ServiceVersion != "" {
			attrs = append(attrs, attribute.String("service.version", cfg.ServiceVersion))
		}
		if cfg.ServiceInstanceID != "" {
			attrs = append(attrs, attribute.String("service.instance.id", cfg.ServiceInstanceID))
		}
	}
	return resource.New(ctx, resource.WithAttributes(attrs...))
}
