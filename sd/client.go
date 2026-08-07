package sd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"

	"github.com/fino-io/gokit/sd/direct"
	"github.com/fino-io/gokit/sd/etcdv3"
)

type Client interface {
	// Register our instance.
	Register(url, service string, tags []string) error

	// Deregister At the end of our service lifecycle, for example at the end of func main,
	// we should make sure to deregister ourselves. This is important! Don't
	// accidentally skip this step by invoking a log.Fatal or os.Exit in the
	// interim, which bypasses the defer stack.
	Deregister() error

	// Instancer It's likely that we'll also want to connect to other services and call
	// their methods. We can build an Instancer to listen for changes from sd,
	// create Endpointer, wrap it with a load-balancer to pick a single
	// endpoint, and finally wrap it with a retry strategy to get something that
	// can be used as an endpoint directly.
	Instancer(service string) sd.Instancer
}

const (
	EtcdMode   = "etcd"
	DirectMode = "direct"
)

var (
	errNilConfig       = errors.New("sd: nil config")
	errEtcdURLRequired = errors.New("sd: etcd url is required")
)

func New(cfg *Config, logger log.Logger) (Client, error) {
	if cfg == nil {
		return nil, errNilConfig
	}

	mode := strings.ToLower(strings.TrimSpace(cfg.Mode))
	if mode == "" {
		mode = inferMode(cfg)
	}

	switch mode {
	case EtcdMode:
		urls := splitURLs(cfg.Url)
		if len(urls) == 0 {
			return nil, errEtcdURLRequired
		}
		if cfg.EtcdV3 == nil {
			cfg.EtcdV3 = &etcdv3.Config{}
		}
		return etcdv3.New(urls, cfg.EtcdV3, logger)
	case DirectMode:
		return direct.New(cfg.Direct), nil
	default:
		return nil, fmt.Errorf("sd: unsupported mode %q", cfg.Mode)
	}
}

func inferMode(cfg *Config) string {
	if len(cfg.Direct) > 0 {
		return DirectMode
	}
	if cfg.EtcdV3 != nil || cfg.Url != "" {
		return EtcdMode
	}
	return ""
}

func splitURLs(raw string) []string {
	parts := strings.Split(raw, ";")
	urls := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			urls = append(urls, part)
		}
	}
	return urls
}
