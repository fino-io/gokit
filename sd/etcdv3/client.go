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
	client    etcdv3.Client
	registrar *etcdv3.Registrar
	logger    log.Logger
	mtx       sync.Mutex
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

	client, err := etcdv3.NewClient(context.Background(), urls, options)
	if err != nil {
		return nil, fmt.Errorf("sd/etcdv3: create client: %w", err)
	}

	return &Client{
		client: client,
		logger: logger,
	}, nil
}

func (c *Client) Register(urlStr, name string, tags []string) error {
	if c == nil || c.client == nil {
		return errors.New("sd/etcdv3: nil client")
	}

	service, err := newService(urlStr, name)
	if err != nil {
		return err
	}

	registrar := etcdv3.NewRegistrar(c.client, service, c.logger)

	c.mtx.Lock()
	defer c.mtx.Unlock()
	if c.registrar != nil {
		c.registrar.Deregister()
	}
	registrar.Register()
	c.registrar = registrar
	return nil
}

func (c *Client) Deregister() error {
	if c == nil {
		return nil
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()
	if c.registrar != nil {
		c.registrar.Deregister()
		c.registrar = nil
	}
	return nil
}

func (c *Client) Instancer(service string) kitsd.Instancer {
	service = strings.Trim(strings.TrimSpace(service), "/")
	if c == nil || c.client == nil || service == "" {
		return nil
	}

	instancer, err := etcdv3.NewInstancer(c.client, "/"+service+"/", c.logger)
	if err != nil {
		_ = c.logger.Log("msg", "create etcd instancer failed", "service", service, "err", err)
		return nil
	}
	return instancer
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
