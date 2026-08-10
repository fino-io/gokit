package etcdv3

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	kitsd "github.com/go-kit/kit/sd"
	"github.com/go-kit/kit/sd/etcdv3"
)

type Client struct {
	registry etcdv3.Client
	urls     []string
	options  etcdv3.ClientOptions
	service  etcdv3.Service
	logger   log.Logger
	mtx      sync.Mutex
}

func New(urls []string, cfg *Config, logger log.Logger) (*Client, error) {
	if len(urls) == 0 {
		return nil, errors.New("sd/etcdv3: urls are required")
	}

	if cfg == nil {
		cfg = &Config{}
	}
	if logger == nil {
		logger = log.NewNopLogger()
	}

	options := etcdv3.ClientOptions{
		// Path to trusted ca file
		CACert: cfg.CACert,
		// Path to certificate
		Cert: cfg.Cert,
		// Path to private key
		Key: cfg.Key,
		// Username if required
		Username: cfg.Username,
		// Password if required
		Password: cfg.Password,
		// If DialTimeout is 0, it defaults to 3s
		DialTimeout: time.Second * time.Duration(cfg.DialTimeout),
		// If DialKeepAlive is 0, it defaults to 3s
		DialKeepAlive: time.Second * time.Duration(cfg.DialKeepAlive),
	}

	registry, err := etcdv3.NewClient(context.Background(), urls, options)
	if err != nil {
		return nil, fmt.Errorf("sd/etcdv3: create registry client: %w", err)
	}

	return &Client{
		registry: registry,
		urls:     append([]string(nil), urls...),
		options:  options,
		logger:   logger,
	}, nil
}

func (c *Client) Register(urlStr, name string) error {
	if c == nil || c.registry == nil {
		return errors.New("sd/etcdv3: nil client")
	}

	service, err := newService(urlStr, name)
	if err != nil {
		return err
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.service.Key != "" {
		if err := c.registry.Deregister(c.service); err != nil {
			return fmt.Errorf("sd/etcdv3: deregister previous service: %w", err)
		}
		c.service = etcdv3.Service{}
	}

	if err := c.registry.Register(service); err != nil {
		return fmt.Errorf("sd/etcdv3: register service: %w", err)
	}
	c.service = service
	return nil
}

func (c *Client) Deregister() error {
	if c == nil {
		return nil
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()
	if c.service.Key != "" {
		if err := c.registry.Deregister(c.service); err != nil {
			return fmt.Errorf("sd/etcdv3: deregister service: %w", err)
		}
		c.service = etcdv3.Service{}
	}
	return nil
}

func (c *Client) Instancer(service string) (kitsd.Instancer, error) {
	service = strings.Trim(strings.TrimSpace(service), "/")
	if c == nil {
		return nil, errors.New("sd/etcdv3: nil client")
	}

	if service == "" {
		return nil, errors.New("sd/etcdv3: service name is required")
	}

	discovery, err := etcdv3.NewClient(context.Background(), c.urls, c.options)
	if err != nil {
		return nil, fmt.Errorf("sd/etcdv3: create discovery client: %w", err)
	}

	instancer, err := etcdv3.NewInstancer(discovery, "/"+service+"/", c.logger)
	if err != nil {
		return nil, fmt.Errorf("sd/etcdv3: create instancer: %w", err)
	}
	return instancer, nil
}

func newService(rawURL, name string) (etcdv3.Service, error) {
	name = strings.Trim(strings.TrimSpace(name), "/")
	if name == "" {
		return etcdv3.Service{}, errors.New("sd/etcdv3: service name is required")
	}

	value := strings.TrimSpace(rawURL)
	if value == "" {
		return etcdv3.Service{}, errors.New("sd/etcdv3: instance url is required")
	}

	key, err := serviceKey(value, name)
	if err != nil {
		return etcdv3.Service{}, err
	}

	return etcdv3.Service{
		Key:   key,
		Value: value,
	}, nil
}

func serviceKey(value, name string) (string, error) {
	parseValue := value
	if !strings.Contains(parseValue, "://") {
		parseValue = "sd://" + parseValue
	}

	u, err := url.Parse(parseValue)
	if err != nil {
		return "", fmt.Errorf("sd/etcdv3: parse instance url: %w", err)
	}

	host := strings.TrimSpace(u.Host)
	if host == "" {
		return "", fmt.Errorf("sd/etcdv3: invalid instance url %q", value)
	}
	return "/" + name + "/" + host, nil
}
