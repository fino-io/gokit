package sd

import (
	"errors"
	"fmt"
	"net"
	"strings"

	kitsd "github.com/go-kit/kit/sd"

	"github.com/fino-io/gokit/sd/direct"
	"github.com/fino-io/gokit/sd/etcdv3"
	"github.com/fino-io/gokit/util/host"
)

type Registrar interface {
	Register(url, service string) error
	Deregister() error
}

type Discovery interface {
	Instancer(service string) (kitsd.Instancer, error)
}

type Client struct {
	Registrar Registrar
	Discovery Discovery
	transport string
}

const (
	EtcdMode   = "etcd"
	DirectMode = "direct"
)

var (
	errNilConfig       = errors.New("sd: nil config")
	errEtcdURLRequired = errors.New("sd: etcd url is required")
)

func New(cfg *Config) (*Client, error) {
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
		client, err := etcdv3.New(urls, cfg.EtcdV3, newLogger())
		if err != nil {
			return nil, err
		}
		return &Client{
			Registrar: client,
			Discovery: client,
			transport: strings.ToLower(strings.TrimSpace(cfg.Transport)),
		}, nil
	case DirectMode:
		return &Client{Discovery: direct.New(cfg.Direct)}, nil
	default:
		return nil, fmt.Errorf("sd: unsupported mode %q", cfg.Mode)
	}
}

func (c *Client) RegisterService(service, httpAddr, grpcAddr string) error {
	if c == nil || c.Registrar == nil {
		return nil
	}

	addr := grpcAddr
	if c.transport == "http" {
		addr = httpAddr
	}

	port, err := portFromAddress(addr)
	if err != nil {
		return err
	}

	instance := net.JoinHostPort(host.Address(), port)
	if err := c.Registrar.Register(instance, service); err != nil {
		return fmt.Errorf("sd: register service %q at %q: %w", service, instance, err)
	}
	return nil
}

func (c *Client) Deregister() error {
	if c == nil || c.Registrar == nil {
		return nil
	}
	return c.Registrar.Deregister()
}

func portFromAddress(addr string) (string, error) {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("sd: invalid server address %q: %w", addr, err)
	}
	if port == "" {
		return "", fmt.Errorf("sd: invalid server address %q: empty port", addr)
	}
	return port, nil
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
