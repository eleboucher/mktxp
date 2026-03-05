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
		labelKeysWithRouter := append([]string{"routerboard_name"}, labelKeys...)
		labelVals := []string{e.RouterID["routerboard_name"], rec["name"]}

		metricMap := map[string]struct {
			name       string
			help       string
			parseFloat bool
		}{
			"memory":      {"container_memory", "Container allocated memory in MB", true},
			"cpu-weight":  {"container_cpu_weight", "Container CPU weight allocation", true},
			"cpu-quota":   {"container_cpu_quota", "Container CPU quota percentage", true},
			"network":     {"container_network", "Container network configuration", false},
			"environment": {"container_environment", "Container environment variables configured", false},
			"volume":      {"container_volume", "Container volume mounts configured", false},
		}

		for key, meta := range metricMap {
			if val, ok := rec[key]; ok && val != "" {
				var value float64
				if meta.parseFloat {
					value = ParseFloat(val)
				} else {
					value = 1
				}
				mb.GaugeVal(ch, meta.name, meta.help, value, labelKeysWithRouter, labelVals)
			}
		}

		if status, ok := rec["running"]; ok {
			containerStatus := 0.0
			if status == trueStr && rec["disabled"] != trueStr {
				containerStatus = 1
			}
			mb.GaugeVal(ch, "container_status", "Container status (1=running, 0=stopped)", containerStatus, labelKeysWithRouter, labelVals)
		}

		if _, ok := rec["disabled"]; ok {
			disabled := 0.0
			if rec["disabled"] == trueStr {
				disabled = 1
			}
			mb.GaugeVal(ch, "container_disabled", "Container disabled status", disabled, labelKeysWithRouter, labelVals)
		}

		infoFields := map[string]string{
			"image":          "Container image",
			"command":        "Container command",
			"entrypoint":     "Container entrypoint",
			"restart-policy": "Container restart policy",
		}

		for key, help := range infoFields {
			if _, ok := rec[key]; ok && rec[key] != "" {
				mb.GaugeVal(ch, "container_"+strings.ReplaceAll(key, "-", "_"), help, 1, labelKeysWithRouter, labelVals)
			}
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "container_info", "Information about container configuration",
				[]string{"name", "image"},
				rec)
		}
	}

	return nil
}
