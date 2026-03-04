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

		getVals := func(keys []string) []string {
			vals := make([]string, len(keys))
			for i, k := range keys {
				vals[i] = rec[k]
			}
			return vals
		}

		mb.Info(ch, "bgp_sessions", "BGP sessions info", labelKeys, rec)

		established := 0.0
		if rec["established"] == "true" {
			established = 1.0
		}
		mb.GaugeVal(ch, "bgp_established", "BGP established", established, labelKeys, getVals(labelKeys))

		if val := rec["uptime"]; val != "" {
			uptimeSeconds := float64(utils.ParseMktUptime(val))
			mb.GaugeVal(ch, "bgp_uptime", "BGP uptime in milliseconds", uptimeSeconds*1000, labelKeys, getVals(labelKeys))
		}

		mb.Gauge(ch, "bgp_prefix_count", "BGP prefix count", "prefix_count", labelKeys, rec)

		mb.Counter(ch, "bgp_remote_messages", "Number of remote messages", "remote_messages", labelKeys, rec)
		mb.Counter(ch, "bgp_local_messages", "Number of local messages", "local_messages", labelKeys, rec)
		mb.Counter(ch, "bgp_remote_bytes", "Number of remote bytes", "remote_bytes", labelKeys, rec)
		mb.Counter(ch, "bgp_local_bytes", "Number of local bytes", "local_bytes", labelKeys, rec)
	}

	return nil
}
