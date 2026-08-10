package direct

import (
	"reflect"
	"testing"

	kitsd "github.com/go-kit/kit/sd"
)

func TestInstancerReturnsCleanCopy(t *testing.T) {
	cfg := map[string]*Config{
		"users": {Urls: []string{" http://127.0.0.1:8080 ", "", "grpc://127.0.0.1:9000"}},
	}
	client := New(cfg)
	cfg["users"].Urls[0] = "changed"

	want := []string{"grpc://127.0.0.1:9000", "http://127.0.0.1:8080"}
	if have := instances(t, client, "users"); !reflect.DeepEqual(want, have) {
		t.Fatalf("want %v, got %v", want, have)
	}
}

func TestRegisterAndDeregister(t *testing.T) {
	client := New(map[string]*Config{
		"users": {Urls: []string{"http://127.0.0.1:8080"}},
	})

	if err := client.Register(" http://127.0.0.1:8081 ", "users", nil); err != nil {
		t.Fatal(err)
	}
	want := []string{"http://127.0.0.1:8080", "http://127.0.0.1:8081"}
	if have := instances(t, client, "users"); !reflect.DeepEqual(want, have) {
		t.Fatalf("want %v, got %v", want, have)
	}

	if err := client.Deregister(); err != nil {
		t.Fatal(err)
	}
	want = kitsd.FixedInstancer{"http://127.0.0.1:8080"}
	if have := instances(t, client, "users"); !reflect.DeepEqual(want, have) {
		t.Fatalf("want %v, got %v", want, have)
	}
}

func TestDeregisterKeepsStaticDuplicate(t *testing.T) {
	client := New(map[string]*Config{
		"users": {Urls: []string{"http://127.0.0.1:8080"}},
	})

	if err := client.Register("http://127.0.0.1:8080", "users", nil); err != nil {
		t.Fatal(err)
	}
	if err := client.Deregister(); err != nil {
		t.Fatal(err)
	}

	want := []string{"http://127.0.0.1:8080"}
	if have := instances(t, client, "users"); !reflect.DeepEqual(want, have) {
		t.Fatalf("want %v, got %v", want, have)
	}
}

func TestRegisterValidatesInput(t *testing.T) {
	client := New(nil)
	if err := client.Register("", "users", nil); err == nil {
		t.Fatal("expected error")
	}
	if err := client.Register("http://127.0.0.1:8080", "", nil); err == nil {
		t.Fatal("expected error")
	}

	var nilClient *Client
	if err := nilClient.Register("http://127.0.0.1:8080", "users", nil); err == nil {
		t.Fatal("expected error")
	}
	if err := nilClient.Deregister(); err != nil {
		t.Fatal(err)
	}

	if err := client.Register("http://127.0.0.1:8080", "users", nil); err != nil {
		t.Fatal(err)
	}
	if err := client.Register("http://127.0.0.1:8081", "orders", nil); err != nil {
		t.Fatal(err)
	}
	if have := instances(t, client, "users"); len(have) != 0 {
		t.Fatalf("expected previous registration removed, got %v", have)
	}
}

func TestRegisterWorksOnZeroValueClient(t *testing.T) {
	var client Client
	if err := client.Register("http://127.0.0.1:8080", "users", nil); err != nil {
		t.Fatal(err)
	}

	want := []string{"http://127.0.0.1:8080"}
	if have := instances(t, &client, "users"); !reflect.DeepEqual(want, have) {
		t.Fatalf("want %v, got %v", want, have)
	}
}

func TestInstancerReceivesUpdates(t *testing.T) {
	client := New(map[string]*Config{"users": {Urls: []string{"http://127.0.0.1:8080"}}})
	instancer := client.Instancer("users")
	events := make(chan kitsd.Event, 2)
	instancer.Register(events)
	<-events

	if err := client.Register("http://127.0.0.1:8081", "users", nil); err != nil {
		t.Fatal(err)
	}
	event := <-events
	if want := []string{"http://127.0.0.1:8080", "http://127.0.0.1:8081"}; !reflect.DeepEqual(want, event.Instances) {
		t.Fatalf("want %v, got %v", want, event.Instances)
	}
}

func instances(t *testing.T, client *Client, service string) []string {
	t.Helper()
	instancer := client.Instancer(service)
	if instancer == nil {
		return nil
	}
	events := make(chan kitsd.Event, 1)
	instancer.Register(events)
	event := <-events
	instancer.Deregister(events)
	return event.Instances
}
