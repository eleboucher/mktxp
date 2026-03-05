package collector

import (
	"context"
	"log/slog"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/eleboucher/mktxp/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

// SystemResourceCollector collects system resource metrics from RouterOS.
type SystemResourceCollector struct{}

func NewSystemResourceCollector() *SystemResourceCollector {
	return &SystemResourceCollector{}
}

func (c *SystemResourceCollector) Name() string { return "system_resource" }

// Describe intentionally sends nothing — unchecked collector with dynamic labels.
func (c *SystemResourceCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *SystemResourceCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	records, err := e.APIConn.Run(
		ctx,
		"/system/resource/print",
		"=.proplist=uptime,version,free-memory,total-memory,cpu,cpu-count,cpu-frequency,cpu-load,free-hdd-space,total-hdd-space,architecture-name,board-name",
	)
	if err != nil {
		slog.Error("system resource collect failed", "router", e.RouterName, "err", err)
		return nil
	}
	if len(records) == 0 {
		return nil
	}

	record := TrimRecord(records[0], []string{
		"uptime", "version", "free_memory", "total_memory",
		"cpu", "cpu_count", "cpu_frequency", "cpu_load",
		"free_hdd_space", "total_hdd_space", "architecture_name", "board_name",
	})

	mb := NewMetricBuilder(e)

	sharedLabels := []string{"version", "board_name", "cpu", "architecture_name"}
	sharedVals := []string{record["version"], record["board_name"], record["cpu"], record["architecture_name"]}

	mb.GaugeVal(ch, "system_uptime", "System uptime in seconds",
		float64(utils.ParseMktUptime(record["uptime"])),
		sharedLabels, sharedVals,
	)
	mb.Gauge(ch, "system_free_memory", "Unused amount of RAM", "free_memory", sharedLabels, record)
	mb.Gauge(ch, "system_total_memory", "Amount of installed RAM", "total_memory", sharedLabels, record)
	mb.Gauge(ch, "system_free_hdd_space", "Free space on hard drive or NAND", "free_hdd_space", sharedLabels, record)
	mb.Gauge(ch, "system_total_hdd_space", "Size of the hard drive or NAND", "total_hdd_space", sharedLabels, record)
	mb.Gauge(ch, "system_cpu_load", "Percentage of used CPU resources", "cpu_load", sharedLabels, record)
	mb.Gauge(ch, "system_cpu_count", "Number of CPUs present on the system", "cpu_count", sharedLabels, record)
	mb.Gauge(ch, "system_cpu_frequency", "Current CPU frequency", "cpu_frequency", sharedLabels, record)

	if e.ConfigEntry.CheckForUpdates {
		curVersion := record["version"]
		if curVersion != "" {
			mb.GaugeVal(ch, "update_available", "Is there a newer version available", 0, []string{"newest_version"}, []string{curVersion})
		}
	}

	return nil
}
