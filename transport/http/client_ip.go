package http

import (
	"net"
	stdhttp "net/http"

	"github.com/realclientip/realclientip-go"
)

type clientIPResolver struct {
	trusted   []net.IPNet
	forwarded realclientip.Strategy
	remote    realclientip.RemoteAddrStrategy
}

func newClientIPResolver(trustedProxies []string) (*clientIPResolver, error) {
	resolver := &clientIPResolver{}
	if len(trustedProxies) == 0 {
		return resolver, nil
	}

	trusted, err := realclientip.AddressesAndRangesToIPNets(trustedProxies...)
	if err != nil {
		return nil, err
	}
	forwarded, err := realclientip.NewRightmostTrustedRangeStrategy("X-Forwarded-For", trusted)
	if err != nil {
		return nil, err
	}
	resolver.trusted = trusted
	resolver.forwarded = forwarded
	return resolver, nil
}

// ClientIP returns the direct peer IP unless the peer is a configured trusted
// proxy, in which case it resolves the rightmost untrusted X-Forwarded-For IP.
func (c *Config) ClientIP(r *stdhttp.Request) string {
	if r == nil {
		return ""
	}
	if c == nil || c.clientIPs == nil {
		return realclientip.RemoteAddrStrategy{}.ClientIP(r.Header, r.RemoteAddr)
	}
	return c.clientIPs.ClientIP(r)
}

func (r *clientIPResolver) ClientIP(req *stdhttp.Request) string {
	peerIP := r.remote.ClientIP(req.Header, req.RemoteAddr)
	if r.forwarded == nil || !containsIP(r.trusted, net.ParseIP(peerIP)) {
		return peerIP
	}
	if forwardedIP := r.forwarded.ClientIP(req.Header, req.RemoteAddr); forwardedIP != "" {
		return forwardedIP
	}
	return peerIP
}

func containsIP(ranges []net.IPNet, ip net.IP) bool {
	if ip == nil {
		return false
	}
	for _, trusted := range ranges {
		if trusted.Contains(ip) {
			return true
		}
	}
	return false
}
