package sd

import (
	"errors"
	"testing"

	"github.com/fino-io/gokit/sd/direct"
)

func TestNewDirectDoesNotRequireTopLevelURL(t *testing.T) {
	client, err := New(&Config{
		Mode: DirectMode,
		Direct: map[string]*direct.Config{
			"users": {Urls: []string{" http://127.0.0.1:8080 "}},
		},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal("expected client")
	}
}

func TestNewEtcdRequiresURL(t *testing.T) {
	_, err := New(&Config{Mode: EtcdMode}, nil)
	if !errors.Is(err, errEtcdURLRequired) {
		t.Fatalf("expected %v, got %v", errEtcdURLRequired, err)
	}
}
