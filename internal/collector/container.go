package collector

import (
	"context"
	"log/slog"
	"strings"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type ContainerCollector struct{}

func NewContainerCollector() *ContainerCollector { return &ContainerCollector{} }

func (c *ContainerCollector) Name() string { return "container" }

func (c *ContainerCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *ContainerCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.Container {
		return nil
	}

	records, err := e.APIConn.Run(ctx, "/container/print")
	if err != nil {
		slog.Debug("container collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	mb := NewMetricBuilder(e)
	labelKeys := []string{"name"}

	for _, raw := range records {
		rec := TrimRecord(raw, nil)
		if rec["name"] == "" {
			continue
		}

		rec["name"] = FormatInterfaceName(rec["name"], "", e.ConfigEntry.InterfaceNameFormat)
		labelKeysWithRouter := append([]string{"router_id"}, labelKeys...)

		mb.GaugeVal(ch, "container_status", "Container status (1=running, 0=stopped)", func() float64 {
			if rec["running"] == "true" && rec["disabled"] != "true" {
				return 1
			}
			return 0
		}(), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"]})

		if _, ok := rec["disabled"]; ok {
			disabled := 0.0
			if rec["disabled"] == "true" {
				disabled = 1
			}
			mb.GaugeVal(ch, "container_disabled", "Container disabled status", disabled, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"]})
		}

		if _, ok := rec["memory"]; ok && rec["memory"] != "" {
			mb.GaugeVal(ch, "container_memory", "Container allocated memory in MB", ParseFloat(rec["memory"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"]})
		}

		if _, ok := rec["cpu-weight"]; ok && rec["cpu-weight"] != "" {
			mb.GaugeVal(ch, "container_cpu_weight", "Container CPU weight allocation", ParseFloat(rec["cpu-weight"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"]})
		}

		if _, ok := rec["cpu-quota"]; ok && rec["cpu-quota"] != "" {
			mb.GaugeVal(ch, "container_cpu_quota", "Container CPU quota percentage", ParseFloat(rec["cpu-quota"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"]})
		}

		if _, ok := rec["network"]; ok && rec["network"] != "" {
			mb.GaugeVal(ch, "container_network", "Container network configuration", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"]})
		}

		if _, ok := rec["environment"]; ok && rec["environment"] != "" {
			mb.GaugeVal(ch, "container_environment", "Container environment variables configured", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"]})
		}

		if _, ok := rec["volume"]; ok && rec["volume"] != "" {
			mb.GaugeVal(ch, "container_volume", "Container volume mounts configured", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"]})
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "container_info", "Information about container configuration",
				[]string{"name", "image"},
				rec)
		}

		for _, metric := range []struct{ name, key string }{
			{"container_image", "image"},
			{"container_command", "command"},
			{"container_entrypoint", "entrypoint"},
			{"container_restart_policy", "restart-policy"},
		} {
			if val, ok := rec[metric.key]; ok && val != "" {
				mb.GaugeVal(ch, metric.name, "Container "+strings.ToUpper(metric.key), 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"]})
			}
		}
	}

	return nil
}
