package collector

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

// IdentityCollector collects system identity metrics from RouterOS.
type IdentityCollector struct{}

func NewIdentityCollector() *IdentityCollector {
	return &IdentityCollector{}
}

func (c *IdentityCollector) Name() string { return "identity" }

// Describe intentionally sends nothing — unchecked collector with dynamic labels.
func (c *IdentityCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *IdentityCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	records, err := e.APIConn.Run(ctx,
		"/system/identity/print",
		"=.proplist=name",
	)
	if err != nil {
		slog.Error("identity collect failed", "router", e.RouterName, "err", err)
		return fmt.Errorf("identity: %w", err)
	}
	if len(records) == 0 {
		return nil
	}

	record := TrimRecord(records[0], []string{"name"})

	mb := NewMetricBuilder(e)
	mb.Info(ch, "system_identity", "System identity", []string{"name"}, record)

	return nil
}
