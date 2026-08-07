package http

import (
	"context"
	"time"

	"github.com/fino-io/core/go/fino/core"
)

//go:generate mockgen -destination=mocks/http.go -package=mocks . IClient
type IClient interface {
	DoHTTPRequest(ctx context.Context, requestParam *RequestParam) error
}

type RequestParam struct {
	RequestURI string
	Method     string
	Header     map[string]string
	Body       any
	Response   *core.Object

	Timeout time.Duration
	Cluster *string
	WithSD  *bool
}
