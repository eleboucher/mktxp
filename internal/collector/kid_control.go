package collector

import (
	"context"
	"log/slog"
	"strings"

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

		mb.GaugeVal(ch, "kid_control_status", "Kid Control status (1=active, 0=inactive)", func() float64 {
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
			mb.GaugeVal(ch, "kid_control_disabled", "Kid Control disabled status", disabled, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"]})
		}

		if _, ok := rec["type"]; ok && rec["type"] != "" {
			mb.GaugeVal(ch, "kid_control_type", "Kid Control type (assigned/dynamic)", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"]})
		}

		if _, ok := rec["max-clients"]; ok && rec["max-clients"] != "" {
			mb.GaugeVal(ch, "kid_control_max_clients", "Kid Control maximum clients limit", ParseFloat(rec["max-clients"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"]})
		}

		if _, ok := rec["current-clients"]; ok && rec["current-clients"] != "" {
			mb.GaugeVal(ch, "kid_control_current_clients", "Kid Control current connected clients count", ParseFloat(rec["current-clients"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"]})
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "kid_control_info", "Information about Kid Control configuration",
				[]string{"name", "type"},
				rec)
		}

		for _, metric := range []struct{ name, key string }{
			{"kid_control_enabled", "enabled"},
			{"kid_control_interface", "interface"},
			{"kid_control_timeout", "timeout"},
		} {
			if val, ok := rec[metric.key]; ok && val != "" {
				mb.GaugeVal(ch, metric.name, "Kid Control "+strings.ToUpper(metric.key), 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"]})
			}
		}
	}

	return nil
}
