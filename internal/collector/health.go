package collector

import (
	"context"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type HealthCollector struct{}

func NewHealthCollector() *HealthCollector {
	return &HealthCollector{}
}

func (c *HealthCollector) Name() string { return "health" }

// Describe intentionally sends nothing — unchecked collector with dynamic labels.
func (c *HealthCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *HealthCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	mb := NewMetricBuilder(e)
	value := 0.0
	if e.APIConn.IsConnected() {
		value = 1
	}
	mb.GaugeVal(ch, "health_up", "Indicates if the router is reachable and responding", value, nil, nil)
	return nil
}
