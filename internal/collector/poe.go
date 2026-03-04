package collector

import (
	"context"
	"log/slog"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type POECollector struct{}

func NewPOECollector() *POECollector                       { return &POECollector{} }
func (c *POECollector) Name() string                       { return "poe" }
func (c *POECollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *POECollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.POE {
		return nil
	}

	records, err := e.APIConn.Run(ctx, "/interface/ethernet/poe/print",
		"=.proplist=name,poe-out,poe-priority,poe-voltage,poe-out-status,poe-out-voltage,poe-out-current,poe-out-power")
	if err != nil {
		slog.Error("poe collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	mb := NewMetricBuilder(e)
	allKeys := []string{"name", "poe_out", "poe_priority", "poe_voltage", "poe_out_status", "poe_out_voltage", "poe_out_current", "poe_out_power"}
	infoKeys := []string{"name", "poe_out", "poe_priority", "poe_voltage", "poe_out_status"}

	for _, raw := range records {
		rec := TrimRecord(raw, allKeys)
		mb.Info(ch, "poe", "POE info metrics", infoKeys, rec)

		if _, ok := rec["poe_out_voltage"]; ok {
			mb.Gauge(ch, "poe_out_voltage", "POE out voltage", "poe_out_voltage", []string{"name"}, rec)
		}
		if _, ok := rec["poe_out_current"]; ok {
			mb.Gauge(ch, "poe_out_current", "POE out current", "poe_out_current", []string{"name"}, rec)
		}
		if _, ok := rec["poe_out_power"]; ok {
			mb.Gauge(ch, "poe_out_power", "POE out power", "poe_out_power", []string{"name"}, rec)
		}
	}

	return nil
}
