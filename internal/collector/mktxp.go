package collector

import (
	"context"
	"log/slog"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type MktxpCollector struct{}

func NewMktxpCollector() *MktxpCollector { return &MktxpCollector{} }

func (c *MktxpCollector) Name() string { return "mktxp" }

func (c *MktxpCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *MktxpCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	records, err := e.APIConn.Run(ctx, "/system/resource/print")
	if err != nil {
		slog.Debug("mktxp collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	mb := NewMetricBuilder(e)
	labelKeysWithRouter := []string{"routerboard_name"}

	for _, raw := range records {
		rec := TrimRecord(raw, nil)
		c.collectSystemMetrics(mb, ch, rec, labelKeysWithRouter, e.RouterID["routerboard_name"])
	}

	return nil
}

func (c *MktxpCollector) collectSystemMetrics(
	mb *MetricBuilder,
	ch chan<- prometheus.Metric,
	rec map[string]string,
	labelKeys []string,
	routerID string,
) {
	metricMap := map[string]struct {
		name       string
		help       string
		parseFloat bool
	}{
		"uptime":            {"system_uptime", "System uptime in seconds", true},
		"version":           {"system_version_info", "System version information", false},
		"memory-total":      {"system_memory_total", "System total memory in MB", true},
		"memory-free":       {"system_memory_free", "System free memory in MB", true},
		"cpu-load":          {"system_cpu_load", "System CPU load percentage", true},
		"free-memory":       {"system_free_memory", "System free memory in MB", true},
		"total-memory":      {"system_total_memory", "System total memory in MB", true},
		"cpu-count":         {"system_cpu_count", "System CPU core count", true},
		"last-start-time":   {"system_last_start_time", "System last start timestamp", true},
		"name":              {"system_name_info", "System hostname information", false},
		"architecture-name": {"system_architecture_info", "System architecture name", false},
		"platform":          {"system_platform_info", "System platform information", false},
		"board-name":        {"system_board_name_info", "System board name information", false},
		"serial-number":     {"system_serial_number_info", "System serial number information", false},
	}

	for key, meta := range metricMap {
		if val, ok := rec[key]; ok && val != "" {
			var value float64
			if meta.parseFloat {
				value = ParseFloat(val)
			} else {
				value = 1.0
			}
			mb.GaugeVal(ch, meta.name, meta.help, value, labelKeys, []string{routerID})
		}
	}

	if comment, ok := rec["comment"]; ok && comment != "" {
		mb.Info(ch, "system_resource_info", "Information about system resources",
			[]string{"name", "version", "architecture"}, rec)
	}
}
