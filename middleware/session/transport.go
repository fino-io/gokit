package session

import (
	"context"
	"net/http"
	"strings"

	gokitsession "github.com/fino-io/gokit/session"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	AuthorizationHeader = "Authorization"
	SessionTokenHeader  = "X-Session-Token"
	SessionTokenCookie  = "session_token"
)

type TransportOption func(*config)

type config struct {
	bearerHeader string
	tokenHeader  string
	tokenCookie  string
}

func WithBearerHeader(name string) TransportOption {
	return func(cfg *config) {
		cfg.bearerHeader = strings.TrimSpace(name)
	}
}

func WithTokenHeader(name string) TransportOption {
	return func(cfg *config) {
		cfg.tokenHeader = strings.TrimSpace(name)
	}
}

func WithTokenCookie(name string) TransportOption {
	return func(cfg *config) {
		cfg.tokenCookie = strings.TrimSpace(name)
	}
}

func HTTPMiddleware(validator gokitsession.Validator, opts ...TransportOption) func(http.Handler) http.Handler {
	cfg := newConfig(opts...)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, err := authenticateHTTP(r.Context(), validator, r, cfg)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UnaryServerInterceptor(validator gokitsession.Validator, opts ...TransportOption) grpc.UnaryServerInterceptor {
	cfg := newConfig(opts...)

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx, err := authenticateMetadata(ctx, validator, cfg)
		if err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

func StreamServerInterceptor(validator gokitsession.Validator, opts ...TransportOption) grpc.StreamServerInterceptor {
	cfg := newConfig(opts...)

	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx, err := authenticateMetadata(stream.Context(), validator, cfg)
		if err != nil {
			return err
		}

		return handler(srv, serverStreamWithContext{ServerStream: stream, ctx: ctx})
	}
}

func authenticateHTTP(ctx context.Context, validator gokitsession.Validator, r *http.Request, cfg config) (context.Context, error) {
	if validator == nil {
		return ctx, gokitsession.ErrValidatorRequired
	}

	token, ok := tokenFromHTTPRequest(r, cfg)
	if !ok {
		return ctx, gokitsession.ErrTokenRequired
	}

	return contextWithValidatedSession(ctx, validator, token)
}

func authenticateMetadata(ctx context.Context, validator gokitsession.Validator, cfg config) (context.Context, error) {
	if validator == nil {
		return ctx, status.Error(codes.Unauthenticated, gokitsession.ErrValidatorRequired.Error())
	}

	token, ok := tokenFromMetadata(ctx, cfg)
	if !ok {
		return ctx, status.Error(codes.Unauthenticated, gokitsession.ErrTokenRequired.Error())
	}

	ctx, err := contextWithValidatedSession(ctx, validator, token)
	if err != nil {
		return ctx, status.Error(codes.Unauthenticated, err.Error())
	}

	return ctx, nil
}

func tokenFromHTTPRequest(r *http.Request, cfg config) (string, bool) {
	if cfg.bearerHeader != "" {
		if token, ok := bearerToken(r.Header.Get(cfg.bearerHeader)); ok {
			return token, true
		}
	}
	if cfg.tokenHeader != "" {
		if token := strings.TrimSpace(r.Header.Get(cfg.tokenHeader)); token != "" {
			return token, true
		}
	}
	if cfg.tokenCookie == "" {
		return "", false
	}
	if cookie, err := r.Cookie(cfg.tokenCookie); err == nil {
		if token := strings.TrimSpace(cookie.Value); token != "" {
			return token, true
		}
	}
	return "", false
}

func tokenFromMetadata(ctx context.Context, cfg config) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}
	if cfg.bearerHeader != "" {
		if token, ok := firstMetadataBearer(md, strings.ToLower(cfg.bearerHeader)); ok {
			return token, true
		}
	}
	if cfg.tokenHeader != "" {
		if token, ok := firstMetadataValue(md, strings.ToLower(cfg.tokenHeader)); ok {
			return token, true
		}
	}
	return "", false
}

func newConfig(opts ...TransportOption) config {
	cfg := config{
		bearerHeader: AuthorizationHeader,
		tokenHeader:  SessionTokenHeader,
		tokenCookie:  SessionTokenCookie,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	return cfg
}

func firstMetadataBearer(md metadata.MD, key string) (string, bool) {
	for _, value := range md.Get(key) {
		if token, ok := bearerToken(value); ok {
			return token, true
		}
	}
	return "", false
}

func firstMetadataValue(md metadata.MD, key string) (string, bool) {
	for _, value := range md.Get(key) {
		if token := strings.TrimSpace(value); token != "" {
			return token, true
		}
	}
	return "", false
}

func bearerToken(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if len(value) < len("Bearer ") || !strings.EqualFold(value[:len("Bearer ")], "Bearer ") {
		return "", false
	}

	token := strings.TrimSpace(value[len("Bearer "):])
	return token, token != ""
}

type serverStreamWithContext struct {
	grpc.ServerStream
	ctx context.Context
}

func (s serverStreamWithContext) Context() context.Context {
	return s.ctx
}
