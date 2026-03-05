package collector

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type ConnectionCollector struct{}

func NewConnectionCollector() *ConnectionCollector                { return &ConnectionCollector{} }
func (c *ConnectionCollector) Name() string                       { return "connection" }
func (c *ConnectionCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *ConnectionCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	mb := NewMetricBuilder(e)

	if e.ConfigEntry.Connections {
		records, err := e.APIConn.Run(ctx, "/ip/firewall/connection/print", "=count-only=")
		if err != nil {
			slog.Error("connection count failed", "router", e.RouterName, "err", err)
		} else if len(records) > 0 {
			mb.GaugeVal(ch, "ip_connections_total", "Number of IP connections",
				ParseFloat(records[0]["ret"]), nil, nil)
		}
	}

	if e.ConfigEntry.ConnectionStats {
		if err := c.collectStats(ctx, e, mb, ch); err != nil {
			slog.Error("connection stats failed", "router", e.RouterName, "err", err)
		}
	}

	return nil
}

func (c *ConnectionCollector) collectStats(ctx context.Context, e *entry.RouterEntry, mb *MetricBuilder, ch chan<- prometheus.Metric) error {
	records, err := e.APIConn.Run(ctx, "/ip/firewall/connection/print",
		"=.proplist=src-address,dst-address,protocol")
	if err != nil {
		return err
	}

	type stats struct {
		count int
		dsts  map[string]struct{}
	}

	byAddr := make(map[string]*stats)
	for _, rec := range records {
		src := strings.SplitN(rec["src-address"], ":", 2)[0]
		dst := fmt.Sprintf("%s(%s)", rec["dst-address"], rec["protocol"])

		if _, ok := byAddr[src]; !ok {
			byAddr[src] = &stats{dsts: make(map[string]struct{})}
		}
		byAddr[src].count++
		byAddr[src].dsts[dst] = struct{}{}
	}

	labelKeys := []string{"src_address", "dst_addresses"}
	for src, s := range byAddr {
		var dstList []string
		for d := range s.dsts {
			dstList = append(dstList, d)
		}
		mb.GaugeVal(ch, "connection_stats", "Open connection stats",
			float64(s.count),
			labelKeys,
			[]string{src, strings.Join(dstList, ", ")},
		)
	}
	return nil
}
