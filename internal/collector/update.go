package collector

import (
	"context"
	"log/slog"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type SystemUpdateCollector struct{}

func NewSystemUpdateCollector() *SystemUpdateCollector { return &SystemUpdateCollector{} }

func (c *SystemUpdateCollector) Name() string { return "system_update" }

func (c *SystemUpdateCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *SystemUpdateCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.CheckForUpdates {
		return nil
	}

	records, err := e.APIConn.Run(ctx, "/system/package/update/print", "=.proplist=status,latest-version,installed-version,channel")
	if err != nil {
		slog.Error("system update info collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	if len(records) == 0 {
		return nil
	}

	mb := NewMetricBuilder(e)
	rec := TrimRecord(records[0], nil)

	updateAvailable := 0.0
	if status, ok := rec["status"]; ok && status == "New version is available" {
		updateAvailable = 1.0
	}

	mb.GaugeVal(ch, "system_update_available", "Is there a newer version available",
		updateAvailable,
		[]string{"newest_version"},
		[]string{rec["latest_version"]},
	)

	return nil
}
