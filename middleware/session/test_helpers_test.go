package session

import (
	"errors"
	"testing"
	"time"

	gokitsession "github.com/fino-io/gokit/session"
	"github.com/stretchr/testify/require"
)

func newTestManager(t *testing.T, store gokitsession.Store, opts ...gokitsession.Option) *gokitsession.Manager {
	t.Helper()

	keyring, err := gokitsession.NewStaticKeyring(gokitsession.Key{
		ID:     "active",
		Secret: []byte("0123456789abcdef0123456789abcdef"),
	})
	require.NoError(t, err)

	codec, err := gokitsession.NewHMACCodec(keyring)
	require.NoError(t, err)

	manager, err := gokitsession.NewManager(store, codec, opts...)
	require.NoError(t, err)

	return manager
}

func testClock(now *time.Time) gokitsession.Option {
	return gokitsession.WithClock(func() time.Time {
		return *now
	})
}

func testIDSequence(ids ...string) gokitsession.Option {
	index := 0
	return gokitsession.WithIDGenerator(func() (string, error) {
		if index >= len(ids) {
			return "", errors.New("no more ids")
		}
		id := ids[index]
		index++
		return id, nil
	})
}
