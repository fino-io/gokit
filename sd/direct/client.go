package direct

import (
	"errors"
	"strings"
	"sync"

	kitsd "github.com/go-kit/kit/sd"

	"github.com/fino-io/gokit/sd/internal/instance"
)

type Client struct {
	mtx        sync.RWMutex
	instances  map[string]*instance.Cache
	registered registration
}

type registration struct {
	service string
	url     string
	added   bool
}

func New(m map[string]*Config) *Client {
	instances := make(map[string]*instance.Cache, len(m))
	for service, cfg := range m {
		if cfg == nil {
			continue
		}
		urls := cleanURLs(cfg.Urls)
		if len(urls) == 0 {
			continue
		}
		instances[service] = newInstancer(urls)
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
		c.instances = make(map[string]*instance.Cache)
	}
	c.deregisterLocked()
	instancer := c.instances[name]
	if instancer == nil {
		instancer = newInstancer(nil)
		c.instances[name] = instancer
	}
	state := instancer.State()
	urls, added := appendUnique(state.Instances, urlStr)
	instancer.Update(kitsd.Event{Instances: urls})
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
	instancer := c.instances[service]
	c.mtx.RUnlock()
	return instancer
}

func (c *Client) deregisterLocked() {
	if c.registered.service == "" {
		return
	}
	if c.registered.added {
		instancer := c.instances[c.registered.service]
		state := instancer.State()
		urls := remove(state.Instances, c.registered.url)
		instancer.Update(kitsd.Event{Instances: urls})
	}
	c.registered = registration{}
}

func newInstancer(urls []string) *instance.Cache {
	instancer := instance.NewCache()
	instancer.Update(kitsd.Event{Instances: urls})
	return instancer
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
