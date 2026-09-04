package metrics

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"
)

var defaultBuckets = prometheus.DefBuckets

func registerOrReuse[T prometheus.Collector](registerer prometheus.Registerer, collector T) T {
	if err := registerer.Register(collector); err != nil {
		var alreadyRegistered prometheus.AlreadyRegisteredError
		if errors.As(err, &alreadyRegistered) {
			existing, ok := alreadyRegistered.ExistingCollector.(T)
			if ok {
				return existing
			}
		}
		panic(err)
	}
	return collector
}
