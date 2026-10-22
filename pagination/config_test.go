package pagination

import (
	"errors"
	"testing"

	fino "github.com/fino-io/finokit/pagination"
)

// Keep the legacy types assignable to the shared pagination types.
var (
	_ *fino.CursorCodec = (*CursorCodec)(nil)
	_ fino.Page[string] = Page[string]{}
	_ Paginator         = fino.Page[string]{}
)

func TestConfiguredCursorCodec(t *testing.T) {
	codec, err := newWithConfig(&config{EncodedKey: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"})
	if err != nil {
		t.Fatal(err)
	}
	if codec == nil {
		t.Fatal("configured codec is nil")
	}
}

func TestConfiguredCursorCodecRejectsNilConfig(t *testing.T) {
	codec, err := newWithConfig(nil)
	if codec != nil || !errors.Is(err, errNilConfig) {
		t.Fatalf("newWithConfig(nil) = (%v, %v), want (nil, %v)", codec, err, errNilConfig)
	}
}
