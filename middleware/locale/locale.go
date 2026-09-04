package locale

import (
	"context"
	"strings"

	"github.com/go-kit/kit/endpoint"
)

type contextKey struct{}

type Resolver interface {
	ResolveLocale(ctx context.Context, req any) (string, bool, error)
}

type ResolverFunc func(ctx context.Context, req any) (string, bool, error)

func (f ResolverFunc) ResolveLocale(ctx context.Context, req any) (string, bool, error) {
	return f(ctx, req)
}

func NewLocaleMW(resolver Resolver) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			if locale, ok := LocaleFromContext(ctx); ok {
				ctx = WithLocale(ctx, locale)
				return next(ctx, req)
			}

			if resolver == nil {
				return next(ctx, req)
			}

			locale, ok, err := resolver.ResolveLocale(ctx, req)
			if err != nil {
				return nil, err
			}
			if ok {
				ctx = WithLocale(ctx, locale)
			}
			return next(ctx, req)
		}
	}
}

func WithLocale(ctx context.Context, locale string) context.Context {
	locale = normalize(locale)
	if len(locale) == 0 {
		return ctx
	}
	return context.WithValue(ctx, contextKey{}, locale)
}

func LocaleFromContext(ctx context.Context) (string, bool) {
	locale, ok := ctx.Value(contextKey{}).(string)
	if !ok || len(locale) == 0 {
		return "", false
	}
	return locale, true
}

func normalize(locale string) string {
	locale = strings.TrimSpace(locale)
	if len(locale) == 0 {
		return ""
	}

	if index := strings.IndexAny(locale, ",;"); index >= 0 {
		locale = locale[:index]
	}

	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(locale)), "_", "-")
}
