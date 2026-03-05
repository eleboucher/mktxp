package collector

import (
	"context"
	"fmt"
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
		return fmt.Errorf("system_update: %w", err)
	}

	if len(records) == 0 {
		return nil
	}

	mb := NewMetricBuilder(e)
	rec := TrimRecord(records[0], nil)

	metricMap := map[string]struct {
		name       string
		help       string
		parseFloat bool
	}{
		"latest_version": {"system_update_available", "Is there a newer version available", false},
	}

	for key, meta := range metricMap {
		if val, ok := rec[key]; ok && val != "" {
			var value float64
			if meta.parseFloat {
				value = ParseFloat(val)
			} else {
				if rec["status"] == "New version is available" {
					value = 1
				} else {
					value = 0
				}
			}
			mb.GaugeVal(ch, meta.name, meta.help, value, []string{"newest_version"}, []string{rec["latest_version"]})
		}
	}

	return nil
}
