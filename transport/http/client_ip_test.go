package http

import (
	stdhttp "net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientIPResolverUsesDirectPeerByDefault(t *testing.T) {
	resolver, err := newClientIPResolver(nil)
	require.NoError(t, err)

	req := &stdhttp.Request{RemoteAddr: "203.0.113.10:1234", Header: stdhttp.Header{}}
	require.Equal(t, "203.0.113.10", resolver.ClientIP(req))
}

func TestClientIPResolverTrustsForwardingHeaderFromConfiguredProxy(t *testing.T) {
	resolver, err := newClientIPResolver([]string{"10.0.0.0/8"})
	require.NoError(t, err)

	req := &stdhttp.Request{
		RemoteAddr: "10.0.0.2:1234",
		Header:     stdhttp.Header{"X-Forwarded-For": []string{"203.0.113.10, 10.0.0.1"}},
	}
	require.Equal(t, "203.0.113.10", resolver.ClientIP(req))
}

func TestClientIPResolverIgnoresForwardingHeaderFromUntrustedPeer(t *testing.T) {
	resolver, err := newClientIPResolver([]string{"10.0.0.0/8"})
	require.NoError(t, err)

	req := &stdhttp.Request{
		RemoteAddr: "203.0.113.10:1234",
		Header:     stdhttp.Header{"X-Forwarded-For": []string{"198.51.100.5"}},
	}
	require.Equal(t, "203.0.113.10", resolver.ClientIP(req))
}

func TestClientIPResolverRejectsInvalidTrustedProxy(t *testing.T) {
	_, err := newClientIPResolver([]string{"invalid"})
	require.Error(t, err)
}
