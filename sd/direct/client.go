package direct

import (
	"errors"
	"strings"

	kitsd "github.com/go-kit/kit/sd"
)

type Client struct {
	instances map[string]kitsd.FixedInstancer
}

func New(m map[string]*Config) *Client {
	instances := make(map[string]kitsd.FixedInstancer, len(m))
	for service, cfg := range m {
		service = strings.TrimSpace(service)
		if service == "" || cfg == nil {
			continue
		}
		urls := cleanURLs(cfg.Urls)
		if len(urls) == 0 {
			continue
		}
		for _, u := range urls {
			instances[service] = appendUnique(instances[service], u)
		}
	}
	return &Client{instances: instances}
}

func (c *Client) Instancer(service string) (kitsd.Instancer, error) {
	if c == nil {
		return nil, errors.New("sd/direct: nil client")
	}

	service = strings.TrimSpace(service)
	instancer, ok := c.instances[service]
	if !ok {
		return nil, errors.New("sd/direct: service not found")
	}

	return append(kitsd.FixedInstancer(nil), instancer...), nil
}

func cleanURLs(raw []string) []string {
	urls := make([]string, 0, len(raw))
	for _, u := range raw {
		if u = strings.TrimSpace(u); u != "" {
			urls = appendUnique(urls, u)
		}
	}
	return urls
}

func appendUnique(urls []string, url string) []string {
	for _, existing := range urls {
		if existing == url {
			return urls
		}
	}
	return append(urls, url)
}
