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
	labelKeysWithRouter := []string{"router_id"}

	for _, raw := range records {
		rec := TrimRecord(raw, nil)

		if _, ok := rec["uptime"]; ok && rec["uptime"] != "" {
			mb.GaugeVal(ch, "system_uptime", "System uptime in seconds", ParseFloat(rec["uptime"]), labelKeysWithRouter, []string{e.RouterID["router_id"]})
		}

		if _, ok := rec["version"]; ok && rec["version"] != "" {
			mb.GaugeVal(ch, "system_version_info", "System version information", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"]})
		}

		if _, ok := rec["memory-total"]; ok && rec["memory-total"] != "" {
			mb.GaugeVal(ch, "system_memory_total", "System total memory in MB", ParseFloat(rec["memory-total"]), labelKeysWithRouter, []string{e.RouterID["router_id"]})
		}

		if _, ok := rec["memory-free"]; ok && rec["memory-free"] != "" {
			mb.GaugeVal(ch, "system_memory_free", "System free memory in MB", ParseFloat(rec["memory-free"]), labelKeysWithRouter, []string{e.RouterID["router_id"]})
		}

		if _, ok := rec["cpu-load"]; ok && rec["cpu-load"] != "" {
			mb.GaugeVal(ch, "system_cpu_load", "System CPU load percentage", ParseFloat(rec["cpu-load"]), labelKeysWithRouter, []string{e.RouterID["router_id"]})
		}

		if _, ok := rec["free-memory"]; ok && rec["free-memory"] != "" {
			mb.GaugeVal(ch, "system_free_memory", "System free memory in MB", ParseFloat(rec["free-memory"]), labelKeysWithRouter, []string{e.RouterID["router_id"]})
		}

		if _, ok := rec["total-memory"]; ok && rec["total-memory"] != "" {
			mb.GaugeVal(ch, "system_total_memory", "System total memory in MB", ParseFloat(rec["total-memory"]), labelKeysWithRouter, []string{e.RouterID["router_id"]})
		}

		if _, ok := rec["cpu-count"]; ok && rec["cpu-count"] != "" {
			mb.GaugeVal(ch, "system_cpu_count", "System CPU core count", ParseFloat(rec["cpu-count"]), labelKeysWithRouter, []string{e.RouterID["router_id"]})
		}

		if _, ok := rec["last-start-time"]; ok && rec["last-start-time"] != "" {
			mb.GaugeVal(ch, "system_last_start_time", "System last start timestamp", ParseFloat(rec["last-start-time"]), labelKeysWithRouter, []string{e.RouterID["router_id"]})
		}

		if _, ok := rec["name"]; ok && rec["name"] != "" {
			mb.GaugeVal(ch, "system_name_info", "System hostname information", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"]})
		}

		if _, ok := rec["architecture-name"]; ok && rec["architecture-name"] != "" {
			mb.GaugeVal(ch, "system_architecture_info", "System architecture name", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"]})
		}

		if _, ok := rec["platform"]; ok && rec["platform"] != "" {
			mb.GaugeVal(ch, "system_platform_info", "System platform information", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"]})
		}

		if _, ok := rec["board-name"]; ok && rec["board-name"] != "" {
			mb.GaugeVal(ch, "system_board_name_info", "System board name information", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"]})
		}

		if _, ok := rec["serial-number"]; ok && rec["serial-number"] != "" {
			mb.GaugeVal(ch, "system_serial_number_info", "System serial number information", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"]})
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "system_resource_info", "Information about system resources",
				[]string{"name", "version", "architecture"},
				rec)
		}
	}

	return nil
}
