package direct

import (
	"reflect"
	"testing"

	kitsd "github.com/go-kit/kit/sd"
)

func TestInstancerReturnsConfiguredInstances(t *testing.T) {
	cfg := map[string]*Config{
		"users": {Urls: []string{" http://127.0.0.1:8080 ", "", "grpc://127.0.0.1:9000"}},
	}
	client := New(cfg)
	cfg["users"].Urls[0] = "changed"

	instancer, err := client.Instancer("users")
	if err != nil {
		t.Fatal(err)
	}
	events := make(chan kitsd.Event, 1)
	instancer.Register(events)
	event := <-events
	if want := []string{"http://127.0.0.1:8080", "grpc://127.0.0.1:9000"}; !reflect.DeepEqual(want, event.Instances) {
		t.Fatalf("want %v, got %v", want, event.Instances)
	}
}

func TestInstancerRequiresKnownService(t *testing.T) {
	if _, err := New(nil).Instancer("users"); err == nil {
		t.Fatal("expected error")
	}

	var client *Client
	if _, err := client.Instancer("users"); err == nil {
		t.Fatal("expected error")
	}
}
