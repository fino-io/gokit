package sd

import (
	"errors"
	"testing"

	"github.com/fino-io/gokit/sd/direct"
)

type recordingRegistrar struct {
	url          string
	service      string
	deregistered bool
}

func (registrar *recordingRegistrar) Register(url, service string) error {
	registrar.url = url
	registrar.service = service
	return nil
}

func (registrar *recordingRegistrar) Deregister() error {
	registrar.deregistered = true
	return nil
}

func TestNewDirectDoesNotRequireTopLevelURL(t *testing.T) {
	client, err := New(&Config{
		Mode: DirectMode,
		Direct: map[string]*direct.Config{
			"users": {Urls: []string{" http://127.0.0.1:8080 "}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal("expected client")
	}
}

func TestNewEtcdRequiresURL(t *testing.T) {
	_, err := New(&Config{Mode: EtcdMode})
	if !errors.Is(err, errEtcdURLRequired) {
		t.Fatalf("expected %v, got %v", errEtcdURLRequired, err)
	}
}

func TestRegisterServiceUsesConfiguredTransport(t *testing.T) {
	t.Setenv("SERVICE_HOST", " service.example ")

	tests := []struct {
		name      string
		transport string
		wantPort  string
	}{
		{name: "http", transport: "http", wantPort: "32101"},
		{name: "grpc", wantPort: "32102"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registrar := &recordingRegistrar{}
			client := &Client{Registrar: registrar, transport: test.transport}

			err := client.RegisterService(
				"identity.v1.UserService",
				"127.0.0.1:32101",
				"127.0.0.1:32102",
			)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := registrar.url, "service.example:"+test.wantPort; got != want {
				t.Fatalf("registered URL = %q, want %q", got, want)
			}
			if got, want := registrar.service, "identity.v1.UserService"; got != want {
				t.Fatalf("registered service = %q, want %q", got, want)
			}
		})
	}
}

func TestRegisterServiceWithoutRegistrarIsNoop(t *testing.T) {
	var client *Client
	if err := client.RegisterService("service", "invalid", "invalid"); err != nil {
		t.Fatal(err)
	}
	if err := client.Deregister(); err != nil {
		t.Fatal(err)
	}
}

func TestRegisterServiceRejectsInvalidAddress(t *testing.T) {
	client := &Client{Registrar: &recordingRegistrar{}}
	if err := client.RegisterService("service", "invalid", "invalid"); err == nil {
		t.Fatal("expected invalid address error")
	}
}

func TestDeregisterDelegatesToRegistrar(t *testing.T) {
	registrar := &recordingRegistrar{}
	client := &Client{Registrar: registrar}

	if err := client.Deregister(); err != nil {
		t.Fatal(err)
	}
	if !registrar.deregistered {
		t.Fatal("expected registrar to be deregistered")
	}
}
