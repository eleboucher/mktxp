package collector

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/eleboucher/mktxp/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

type NetwatchCollector struct{}

func NewNetwatchCollector() *NetwatchCollector                  { return &NetwatchCollector{} }
func (c *NetwatchCollector) Name() string                       { return "netwatch" }
func (c *NetwatchCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *NetwatchCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.Netwatch {
		return nil
	}

	records, err := e.APIConn.Run(ctx, "/tool/netwatch/print",
		"=.proplist=host,timeout,interval,since,status,comment,name,done-tests,type,failed-tests,loss-count,loss-percent,rtt-avg,rtt-min,rtt-max,rtt-jitter,rtt-stdev,tcp-connect-time,http-status-code,http-resp-time",
	)
	if err != nil {
		slog.Error("netwatch collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	wantedKeys := []string{
		"host", "timeout", "interval", "since", "status", "comment", "name",
		"done_tests", "type", "failed_tests", "loss_count", "loss_percent",
		"rtt_avg", "rtt_min", "rtt_max", "rtt_jitter", "rtt_stdev",
		"tcp_connect_time", "http_status_code", "http_resp_time",
	}

	mb := NewMetricBuilder(e)

	var trimmed []map[string]string
	for _, raw := range records {
		rec := TrimRecord(raw, wantedKeys)
		// Translate status: "up" → 1, else 0
		if rec["status"] == "up" {
			rec["status"] = "1"
		} else {
			rec["status"] = "0"
		}
		// Translate RTT fields from RouterOS time strings to milliseconds
		for _, k := range []string{"rtt_avg", "rtt_min", "rtt_max", "rtt_jitter", "rtt_stdev", "tcp_connect_time", "http_resp_time"} {
			if v := rec[k]; v != "" {
				rec[k] = floatStr(float64(utils.ParseTimedelta(v, true)))
			}
		}
		trimmed = append(trimmed, rec)
	}

	if len(trimmed) == 0 {
		return nil
	}

	infoLabels := []string{"host", "timeout", "interval", "since", "status", "comment", "name"}
	for _, rec := range trimmed {
		mb.Info(ch, "netwatch", "Netwatch info metrics", infoLabels, rec)
		mb.Gauge(ch, "netwatch_status", "Netwatch status", "status", []string{"name", "type"}, rec)
		mb.Gauge(ch, "netwatch_done_tests", "Netwatch done tests", "done_tests", []string{"name", "type"}, rec)
		mb.Gauge(ch, "netwatch_failed_tests", "Netwatch failed tests", "failed_tests", []string{"name", "type"}, rec)

		switch rec["type"] {
		case "icmp":
			mb.Gauge(ch, "netwatch_icmp_loss_count", "Netwatch ICMP loss count", "loss_count", []string{"name", "type"}, rec)
			mb.Gauge(ch, "netwatch_icmp_loss_percent", "Netwatch ICMP loss percent", "loss_percent", []string{"name", "type"}, rec)
			mb.Gauge(ch, "netwatch_icmp_rtt_avg_ms", "Netwatch ICMP round trip average", "rtt_avg", []string{"name", "type"}, rec)
			mb.Gauge(ch, "netwatch_icmp_rtt_min_ms", "Netwatch ICMP round trip min", "rtt_min", []string{"name", "type"}, rec)
			mb.Gauge(ch, "netwatch_icmp_rtt_max_ms", "Netwatch ICMP round trip max", "rtt_max", []string{"name", "type"}, rec)
			mb.Gauge(ch, "netwatch_icmp_rtt_jitter_ms", "Netwatch ICMP round trip jitter", "rtt_jitter", []string{"name", "type"}, rec)
			mb.Gauge(ch, "netwatch_icmp_rtt_stdev_ms", "Netwatch ICMP round trip stdev", "rtt_stdev", []string{"name", "type"}, rec)
		case "tcp-conn":
			mb.Gauge(ch, "netwatch_tcp_connect_time_ms", "Netwatch TCP connect time", "tcp_connect_time", []string{"name", "type"}, rec)
		case "http-get", "https-get":
			mb.Gauge(ch, "netwatch_http_status_code", "Netwatch HTTP status code", "http_status_code", []string{"name", "type"}, rec)
			mb.Gauge(ch, "netwatch_http_resp_time", "Netwatch HTTP response time", "http_resp_time", []string{"name", "type"}, rec)
			mb.Gauge(ch, "netwatch_tcp_connect_time_ms", "Netwatch TCP connect time", "tcp_connect_time", []string{"name", "type"}, rec)
		}
	}

	return nil
}

// floatStr converts a float64 to its decimal string representation.
func floatStr(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
