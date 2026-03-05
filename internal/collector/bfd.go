package collector

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type BFDCollector struct{}

func NewBFDCollector() *BFDCollector { return &BFDCollector{} }

func (c *BFDCollector) Name() string { return "bfd" }

func (c *BFDCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *BFDCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.BFD {
		return nil
	}

	records, err := e.APIConn.Run(ctx, "/routing/bfd/peer/print")
	if err != nil {
		slog.Debug("bfd collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	mb := NewMetricBuilder(e)
	labelKeys := []string{"name", "remote_address", "local_interface"}

	for _, raw := range records {
		rec := TrimRecord(raw, nil)
		if rec["name"] == "" {
			continue
		}

		rec["name"] = FormatInterfaceName(rec["name"], "", e.ConfigEntry.InterfaceNameFormat)
		labelKeysWithRouter := append([]string{"router_id"}, labelKeys...)

		mb.GaugeVal(ch, "bfd_session_status", "BFD session status (1=up, 0=down)", func() float64 {
			if rec["state"] == "active" || rec["state"] == "up" {
				return 1
			}
			return 0
		}(), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"], rec["local_interface"]})

		if _, ok := rec["multiplier"]; ok && rec["multiplier"] != "" {
			if v, err := strconv.ParseFloat(rec["multiplier"], 64); err == nil {
				mb.GaugeVal(ch, "bfd_multiplier", "BFD detection multiplier", v, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"], rec["local_interface"]})
			}
		}

		if _, ok := rec["min_tx_interval"]; ok && rec["min_tx_interval"] != "" {
			mb.GaugeVal(ch, "bfd_min_tx_interval", "BFD minimum transmit interval (ms)", ParseFloat(rec["min_tx_interval"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"], rec["local_interface"]})
		}

		if _, ok := rec["min_rx_interval"]; ok && rec["min_rx_interval"] != "" {
			mb.GaugeVal(ch, "bfd_min_rx_interval", "BFD minimum receive interval (ms)", ParseFloat(rec["min_rx_interval"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"], rec["local_interface"]})
		}

		if _, ok := rec["echo_mode"]; ok {
			echoMode := 0.0
			if strings.ToLower(rec["echo_mode"]) == "yes" || rec["echo_mode"] == "true" {
				echoMode = 1
			}
			mb.GaugeVal(ch, "bfd_echo_mode", "BFD echo mode enabled", echoMode, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"], rec["local_interface"]})
		}

		if _, ok := rec["disabled"]; ok {
			disabled := 0.0
			if rec["disabled"] == "true" {
				disabled = 1
			}
			mb.GaugeVal(ch, "bfd_disabled", "BFD session disabled status", disabled, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"], rec["local_interface"]})
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "bfd_session_info", "Information about BFD session",
				[]string{"name", "remote_address", "local_interface", "comment"},
				rec)
		}

		for _, metric := range []struct{ name, key string }{
			{"bfd_state", "state"},
			{"bfd_remote_system_id", "remote-system-id"},
			{"bfd_local_discriminator", "local-discriminator"},
			{"bfd_remote_discriminator", "remote-discriminator"},
		} {
			if val, ok := rec[metric.key]; ok && val != "" {
				mb.GaugeVal(ch, metric.name, "BFD "+strings.ToUpper(metric.key), ParseFloat(val), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"], rec["local_interface"]})
			}
		}
	}

	return nil
}
