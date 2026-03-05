package collector

import (
	"context"
	"log/slog"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/eleboucher/mktxp/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

type BGPCollector struct{}

func NewBGPCollector() *BGPCollector { return &BGPCollector{} }

func (c *BGPCollector) Name() string { return "bgp" }

func (c *BGPCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *BGPCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.BGP {
		return nil
	}

	records, err := e.APIConn.Run(ctx, "/routing/bgp/session/print", "=.proplist=name,remote.address,remote.as,local.address,local.as,established,uptime,prefix-count,remote.messages,local.messages,remote.bytes,local.bytes")
	if err != nil {
		slog.Error("bgp collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	mb := NewMetricBuilder(e)
	labelKeys := []string{"name", "remote_address", "remote_as", "local_address", "local_as"}

	for _, raw := range records {
		rec := TrimRecord(raw, nil)

		mb.Info(ch, "bgp_sessions", "BGP sessions info", labelKeys, rec)

		established := 0.0
		if rec["established"] == trueStr {
			established = 1.0
		}
		mb.GaugeVal(ch, "bgp_established", "BGP established", established, labelKeys, labelKeys)

		labelVals := []string{rec["name"], rec["remote.address"], rec["remote.as"], rec["local.address"], rec["local.as"]}

		metricMap := map[string]struct {
			name       string
			help       string
			parseFloat bool
		}{
			"uptime":          {"bgp_uptime", "BGP uptime in milliseconds", false},
			"prefix_count":    {"bgp_prefix_count", "BGP prefix count", true},
			"remote_messages": {"bgp_remote_messages", "Number of remote messages", true},
			"local_messages":  {"bgp_local_messages", "Number of local messages", true},
			"remote_bytes":    {"bgp_remote_bytes", "Number of remote bytes", true},
			"local_bytes":     {"bgp_local_bytes", "Number of local bytes", true},
		}

		for key, meta := range metricMap {
			if val, ok := rec[key]; ok && val != "" {
				var value float64
				if meta.parseFloat {
					value = ParseFloat(val)
				} else {
					value = float64(utils.ParseMktUptime(val)) * 1000
				}
				mb.GaugeVal(ch, meta.name, meta.help, value, labelKeys, labelVals)
			}
		}
	}

	return nil
}
