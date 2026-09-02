package host

import "testing"

func TestAddressUsesConfiguredServiceHost(t *testing.T) {
	t.Setenv("SERVICE_HOST", " service.example ")

	if got := Address(); got != "service.example" {
		t.Fatalf("Address() = %q, want %q", got, "service.example")
	}
}
