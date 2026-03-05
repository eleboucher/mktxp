package collector

import (
	"context"
	"log/slog"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type KidControlCollector struct{}

func NewKidControlCollector() *KidControlCollector { return &KidControlCollector{} }

func (c *KidControlCollector) Name() string { return "kid_control" }

func (c *KidControlCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *KidControlCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.KidControlAssigned && !e.ConfigEntry.KidControlDynamic {
		return nil
	}

	records, err := e.APIConn.Run(ctx, "/interface/wireless/kid-control/print")
	if err != nil {
		slog.Debug("kid_control collect failed", "router", e.RouterName, "err", err)
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
		labelVals := []string{e.RouterID["router_id"], rec["name"]}

		metricMap := map[string]struct {
			name       string
			help       string
			parseFloat bool
		}{
			"type":            {"kid_control_type", "Kid Control type (assigned/dynamic)", false},
			"max-clients":     {"kid_control_max_clients", "Kid Control maximum clients limit", true},
			"current-clients": {"kid_control_current_clients", "Kid Control current connected clients count", true},
			"enabled":         {"kid_control_enabled", "Kid Control enabled", false},
			"interface":       {"kid_control_interface", "Kid Control interface", false},
			"timeout":         {"kid_control_timeout", "Kid Control timeout", false},
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
			kidControlStatus := 0.0
			if status == trueStr && rec["disabled"] != trueStr {
				kidControlStatus = 1
			}
			mb.GaugeVal(ch, "kid_control_status", "Kid Control status (1=active, 0=inactive)", kidControlStatus, labelKeysWithRouter, labelVals)
		}

		if _, ok := rec["disabled"]; ok {
			disabled := 0.0
			if rec["disabled"] == trueStr {
				disabled = 1
			}
			mb.GaugeVal(ch, "kid_control_disabled", "Kid Control disabled status", disabled, labelKeysWithRouter, labelVals)
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "kid_control_info", "Information about Kid Control configuration",
				[]string{"name", "type"},
				rec)
		}
	}

	return nil
}
