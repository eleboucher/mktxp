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
		if rec["status"] == "up" {
			rec["status"] = "1"
		} else {
			rec["status"] = "0"
		}
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

		labelKeys := []string{"name", "type"}
		labelVals := []string{rec["name"], rec["type"]}

		metricMap := map[string]struct {
			name       string
			help       string
			parseFloat bool
		}{
			"status":           {"netwatch_status", "Netwatch status", false},
			"done_tests":       {"netwatch_done_tests", "Netwatch done tests", true},
			"failed_tests":     {"netwatch_failed_tests", "Netwatch failed tests", true},
			"loss_count":       {"netwatch_icmp_loss_count", "Netwatch ICMP loss count", true},
			"loss_percent":     {"netwatch_icmp_loss_percent", "Netwatch ICMP loss percent", true},
			"rtt_avg":          {"netwatch_icmp_rtt_avg_ms", "Netwatch ICMP round trip average", true},
			"rtt_min":          {"netwatch_icmp_rtt_min_ms", "Netwatch ICMP round trip min", true},
			"rtt_max":          {"netwatch_icmp_rtt_max_ms", "Netwatch ICMP round trip max", true},
			"rtt_jitter":       {"netwatch_icmp_rtt_jitter_ms", "Netwatch ICMP round trip jitter", true},
			"rtt_stdev":        {"netwatch_icmp_rtt_stdev_ms", "Netwatch ICMP round trip stdev", true},
			"tcp_connect_time": {"netwatch_tcp_connect_time_ms", "Netwatch TCP connect time", true},
			"http_status_code": {"netwatch_http_status_code", "Netwatch HTTP status code", true},
			"http_resp_time":   {"netwatch_http_resp_time", "Netwatch HTTP response time", true},
		}

		for key, meta := range metricMap {
			if val, ok := rec[key]; ok && val != "" {
				var value float64
				if meta.parseFloat {
					value = ParseFloat(val)
				} else {
					value = 1
				}
				mb.GaugeVal(ch, meta.name, meta.help, value, labelKeys, labelVals)
			}
		}
	}

	return nil
}

func floatStr(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
