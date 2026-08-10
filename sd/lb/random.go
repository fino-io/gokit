package lb

import (
	"math/rand/v2"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/sd"
)

// NewRandom returns a load balancer that selects services randomly.
func NewRandom(s sd.Endpointer) Balancer {
	return &random{s: s}
}

type random struct {
	s sd.Endpointer
}

func (r *random) Endpoint() (endpoint.Endpoint, error) {
	endpoints, err := r.s.Endpoints()
	if err != nil {
		return nil, err
	}
	if len(endpoints) <= 0 {
		return nil, ErrNoEndpoints
	}

	return endpoints[rand.IntN(len(endpoints))], nil
}
