package pagination

import "testing"

func TestCursorCodecRoundTrip(t *testing.T) {
	codec, err := NewCursorCodec(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	binding, err := binding("users", []int64{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	token, err := codec.EncodeOffset(12, binding)
	if err != nil {
		t.Fatal(err)
	}
	offset, err := codec.DecodeOffset(token, binding)
	if err != nil {
		t.Fatal(err)
	}
	if offset != 12 {
		t.Fatalf("offset = %d, want 12", offset)
	}
}

func TestCursorCodecRejectsDifferentBinding(t *testing.T) {
	codec, err := NewCursorCodec(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	token, err := codec.EncodeOffset(12, "users")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := codec.DecodeOffset(token, "admins"); err != ErrInvalidPageToken {
		t.Fatalf("Decode() error = %v, want %v", err, ErrInvalidPageToken)
	}
}

func TestCursorCodecDecodeOffsetAcceptsEmptyToken(t *testing.T) {
	codec, err := NewCursorCodec(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	binding, err := binding("users", []int64{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	offset, err := codec.DecodeOffset("", binding)
	if err != nil || offset != 0 {
		t.Fatalf("DecodeOffset() = (%d, %v)", offset, err)
	}
}

func TestNormalizePageSize(t *testing.T) {
	for _, test := range []struct {
		input int
		want  int
		valid bool
	}{
		{input: 0, want: DefaultPageSize, valid: true},
		{input: MaxPageSize + 1, want: MaxPageSize, valid: true},
		{input: -1, valid: false},
	} {
		got, err := normalizePageSize(test.input)
		if (err == nil) != test.valid || got != test.want {
			t.Fatalf("NormalizePageSize(%d) = (%d, %v)", test.input, got, err)
		}
	}
}

func TestResolve(t *testing.T) {
	codec, err := NewCursorCodec(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}

	binding, err := binding("users", []int64{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	token, err := codec.EncodeOffset(12, binding)
	if err != nil {
		t.Fatal(err)
	}

	offset, size, gotBinding, err := Resolve(codec, token, 25, "users", []int64{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	if offset != 12 || size != 25 || gotBinding != binding {
		t.Fatalf("Resolve() = (%d, %d, %q), want (12, 25, %q)", offset, size, gotBinding, binding)
	}
}

func TestPageImplementsPaginator(t *testing.T) {
	page := Page[string]{Items: []string{"one"}, TotalCount: 1, NextPageToken: "next"}
	var paginator Paginator = page
	if paginator.GetTotalCount() != 1 || paginator.GetNextPageToken() != "next" {
		t.Fatalf("unexpected paginator: %#v", paginator)
	}
}

func TestCursorCodecFromBase64(t *testing.T) {
	codec, err := newCursorCodecFromBase64("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
	if err != nil || codec == nil {
		t.Fatalf("newCursorCodecFromBase64() = (%v, %v)", codec, err)
	}
	if _, err := newCursorCodecFromBase64("invalid"); err == nil {
		t.Fatal("newCursorCodecFromBase64() error = nil, want error")
	}
}

func TestConfiguredCursorCodec(t *testing.T) {
	codec, err := newWithConfig(&config{EncodedKey: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"})
	if err != nil || codec == nil {
		t.Fatalf("newWithConfig() = (%v, %v)", codec, err)
	}
	if _, err := newWithConfig(nil); err == nil {
		t.Fatal("newWithConfig(nil) error = nil, want error")
	}
}
