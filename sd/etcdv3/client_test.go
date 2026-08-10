package etcdv3

import (
	"errors"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd/etcdv3"
)

func TestNewServicePreservesInstanceValue(t *testing.T) {
	service, err := newService("http://127.0.0.1:8080", "/users/")
	if err != nil {
		t.Fatal(err)
	}
	if service.Key != "/users/127.0.0.1:8080" {
		t.Fatalf("unexpected key %q", service.Key)
	}
	if service.Value != "http://127.0.0.1:8080" {
		t.Fatalf("unexpected value %q", service.Value)
	}
}

func TestNewServiceAcceptsBareHostPort(t *testing.T) {
	service, err := newService("127.0.0.1:8080", "users")
	if err != nil {
		t.Fatal(err)
	}
	if service.Key != "/users/127.0.0.1:8080" {
		t.Fatalf("unexpected key %q", service.Key)
	}
	if service.Value != "127.0.0.1:8080" {
		t.Fatalf("unexpected value %q", service.Value)
	}
}

func TestRegisterReplacesPreviousRegistration(t *testing.T) {
	backend := &fakeClient{}
	client := &Client{registry: backend, logger: log.NewNopLogger()}
	if err := client.Register("http://127.0.0.1:8080", "users"); err != nil {
		t.Fatal(err)
	}
	if err := client.Register("http://127.0.0.1:8081", "users"); err != nil {
		t.Fatal(err)
	}
	if len(backend.deregistered) != 1 || backend.deregistered[0].Key != "/users/127.0.0.1:8080" {
		t.Fatalf("expected first service to be deregistered, got %v", backend.deregistered)
	}

	if err := client.Deregister(); err != nil {
		t.Fatal(err)
	}
	if len(backend.deregistered) != 2 || backend.deregistered[1].Key != "/users/127.0.0.1:8081" {
		t.Fatalf("expected second service to be deregistered, got %v", backend.deregistered)
	}
}

func TestRegisterReturnsBackendError(t *testing.T) {
	backend := &fakeClient{registerErr: errors.New("unavailable")}
	client := &Client{registry: backend, logger: log.NewNopLogger()}
	if err := client.Register("http://127.0.0.1:8080", "users"); !errors.Is(err, backend.registerErr) {
		t.Fatalf("expected %v, got %v", backend.registerErr, err)
	}
}

func TestInstancerRequiresDiscoveryClient(t *testing.T) {
	client := &Client{logger: log.NewNopLogger()}
	if _, err := client.Instancer("users"); err == nil {
		t.Fatal("expected error")
	}
}

type fakeClient struct {
	registered   []etcdv3.Service
	deregistered []etcdv3.Service
	registerErr  error
}

func (c *fakeClient) GetEntries(string) ([]string, error) { return nil, nil }
func (c *fakeClient) WatchPrefix(string, chan struct{})   {}
func (c *fakeClient) Register(service etcdv3.Service) error {
	if c.registerErr != nil {
		return c.registerErr
	}
	c.registered = append(c.registered, service)
	return nil
}
func (c *fakeClient) Deregister(service etcdv3.Service) error {
	c.deregistered = append(c.deregistered, service)
	return nil
}
func (c *fakeClient) LeaseID() int64 { return 0 }
