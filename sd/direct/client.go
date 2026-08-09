package direct

import (
	"errors"
	"strings"
	"sync"

	kitsd "github.com/go-kit/kit/sd"
)

type Client struct {
	mtx        sync.RWMutex
	instances  map[string][]string
	registered registration
}

type registration struct {
	service string
	url     string
	added   bool
}

func New(m map[string]*Config) *Client {
	instances := make(map[string][]string, len(m))
	for service, cfg := range m {
		if cfg == nil {
			continue
		}
		urls := cleanURLs(cfg.Urls)
		if len(urls) == 0 {
			continue
		}
		instances[service] = urls
	}
	return &Client{instances: instances}
}

func (c *Client) Register(urlStr, name string, tags []string) error {
	if c == nil {
		return errors.New("sd/direct: nil client")
	}

	name = strings.TrimSpace(name)
	urlStr = strings.TrimSpace(urlStr)
	if name == "" {
		return errors.New("sd/direct: service name is required")
	}
	if urlStr == "" {
		return errors.New("sd/direct: instance url is required")
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()
	if c.instances == nil {
		c.instances = make(map[string][]string)
	}
	c.deregisterLocked()
	urls, added := appendUnique(c.instances[name], urlStr)
	c.instances[name] = urls
	c.registered = registration{service: name, url: urlStr, added: added}
	return nil
}

func (c *Client) Deregister() error {
	if c == nil {
		return nil
	}

	c.mtx.Lock()
	c.deregisterLocked()
	c.mtx.Unlock()
	return nil
}

func (c *Client) Instancer(service string) kitsd.Instancer {
	if c == nil {
		return nil
	}

	c.mtx.RLock()
	urls := append([]string(nil), c.instances[service]...)
	c.mtx.RUnlock()

	if len(urls) == 0 {
		return nil
	}

	var ret kitsd.FixedInstancer
	return append(ret, urls...)
}

func (c *Client) deregisterLocked() {
	if c.registered.service == "" {
		return
	}
	if c.registered.added {
		urls := remove(c.instances[c.registered.service], c.registered.url)
		if len(urls) == 0 {
			delete(c.instances, c.registered.service)
		} else {
			c.instances[c.registered.service] = urls
		}
	}
	c.registered = registration{}
}

func cleanURLs(raw []string) []string {
	urls := make([]string, 0, len(raw))
	for _, u := range raw {
		if u = strings.TrimSpace(u); u != "" {
			urls, _ = appendUnique(urls, u)
		}
	}
	return urls
}

func appendUnique(urls []string, url string) ([]string, bool) {
	for _, existing := range urls {
		if existing == url {
			return urls, false
		}
	}
	return append(urls, url), true
}

func remove(urls []string, url string) []string {
	for i, existing := range urls {
		if existing == url {
			return append(urls[:i], urls[i+1:]...)
		}
	}
	return urls
}
