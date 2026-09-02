package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func counterValue(t *testing.T, gatherer prometheus.Gatherer, name string, labels map[string]string) float64 {
	t.Helper()
	families, err := gatherer.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.Metric {
			if metricLabels(metric, labels) {
				return metric.GetCounter().GetValue()
			}
		}
	}
	return 0
}

func gaugeValue(t *testing.T, gatherer prometheus.Gatherer, name string, labels map[string]string) float64 {
	t.Helper()
	families, err := gatherer.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.Metric {
			if metricLabels(metric, labels) {
				return metric.GetGauge().GetValue()
			}
		}
	}
	return 0
}

func histogramCount(t *testing.T, gatherer prometheus.Gatherer, name string, labels map[string]string) uint64 {
	t.Helper()
	families, err := gatherer.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.Metric {
			if metricLabels(metric, labels) {
				return metric.GetHistogram().GetSampleCount()
			}
		}
	}
	return 0
}

func metricLabels(metric *dto.Metric, labels map[string]string) bool {
	for key, value := range labels {
		found := false
		for _, pair := range metric.Label {
			if pair.GetName() == key && pair.GetValue() == value {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
