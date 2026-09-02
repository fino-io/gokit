// Package grpc provides shared gRPC client connection construction.
package grpc

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/fino-io/gokit/metrics"
	"github.com/fino-io/gokit/middleware/accesslog"
	"github.com/fino-io/gokit/tracing"
	kitsd "github.com/go-kit/kit/sd"
	"go.opentelemetry.io/otel/trace"
	stdgrpc "google.golang.org/grpc"
	_ "google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const roundRobinServiceConfig = `{"loadBalancingConfig":[{"round_robin":{}}]}`

var errTransportCredentialsRequired = errors.New("grpc: transport credentials are required")

type clientOptions struct {
	credentials   credentials.TransportCredentials
	instancer     kitsd.Instancer
	unary         []stdgrpc.UnaryClientInterceptor
	stream        []stdgrpc.StreamClientInterceptor
	serviceConfig string
	dialOptions   []stdgrpc.DialOption
}

// ClientOption configures a client connection.
type ClientOption func(*clientOptions)

// WithTransportCredentials configures transport security.
func WithTransportCredentials(credentials credentials.TransportCredentials) ClientOption {
	return func(options *clientOptions) {
		options.credentials = credentials
	}
}

// WithInsecure explicitly opts into plaintext transport.
func WithInsecure() ClientOption {
	return WithTransportCredentials(insecure.NewCredentials())
}

// WithInstancer uses a Go-kit Instancer as the connection's name resolver.
// The caller owns the Instancer. Closing the connection deregisters the
// resolver listener but does not call Instancer.Stop.
func WithInstancer(instancer kitsd.Instancer) ClientOption {
	return func(options *clientOptions) {
		options.instancer = instancer
	}
}

// WithUnaryInterceptors appends unary client interceptors in invocation order.
func WithUnaryInterceptors(interceptors ...stdgrpc.UnaryClientInterceptor) ClientOption {
	return func(options *clientOptions) {
		options.unary = append(options.unary, interceptors...)
	}
}

// WithClientObservability adds request ID propagation, tracing, and metrics to unary client calls.
// caller identifies the calling service and target identifies the remote service.
func WithClientObservability(caller, target string, tracer trace.Tracer, instrumentation *metrics.Instrumentation) ClientOption {
	return WithUnaryInterceptors(
		accesslog.UnaryClientInterceptor(),
		tracing.GRPCUnaryClientInterceptor(tracer),
		instrumentation.GRPCUnaryClientInterceptor(caller, target),
	)
}

// WithStreamInterceptors appends stream client interceptors in invocation order.
func WithStreamInterceptors(interceptors ...stdgrpc.StreamClientInterceptor) ClientOption {
	return func(options *clientOptions) {
		options.stream = append(options.stream, interceptors...)
	}
}

// WithDefaultServiceConfig replaces the complete default service config.
// When discovery is enabled, include loadBalancingConfig in custom configs to
// preserve round_robin behavior.
func WithDefaultServiceConfig(serviceConfig string) ClientOption {
	return func(options *clientOptions) {
		options.serviceConfig = strings.TrimSpace(serviceConfig)
	}
}

// WithDialOptions appends advanced native gRPC options. Use the dedicated
// options above for transport credentials, discovery, and interceptors.
func WithDialOptions(options ...stdgrpc.DialOption) ClientOption {
	return func(config *clientOptions) {
		config.dialOptions = append(config.dialOptions, options...)
	}
}

// NewClient constructs a standard gRPC ClientConn. The caller owns the connection.
func NewClient(target string, options ...ClientOption) (*stdgrpc.ClientConn, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, errors.New("grpc: target is required")
	}

	var config clientOptions
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	if config.credentials == nil {
		return nil, errTransportCredentialsRequired
	}

	dialOptions := []stdgrpc.DialOption{stdgrpc.WithTransportCredentials(config.credentials)}
	if len(config.unary) > 0 {
		dialOptions = append(dialOptions, stdgrpc.WithChainUnaryInterceptor(config.unary...))
	}
	if len(config.stream) > 0 {
		dialOptions = append(dialOptions, stdgrpc.WithChainStreamInterceptor(config.stream...))
	}
	if config.instancer != nil {
		builder := newInstancerBuilder(config.instancer)
		dialOptions = append(dialOptions, stdgrpc.WithResolvers(builder))
		target = fmt.Sprintf("%s:///%s", builder.Scheme(), url.PathEscape(target))
		if config.serviceConfig == "" {
			config.serviceConfig = roundRobinServiceConfig
		}
	}
	if config.serviceConfig != "" {
		dialOptions = append(dialOptions, stdgrpc.WithDefaultServiceConfig(config.serviceConfig))
	}
	dialOptions = append(dialOptions, config.dialOptions...)

	return stdgrpc.NewClient(target, dialOptions...)
}
