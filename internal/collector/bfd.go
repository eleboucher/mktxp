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
		labelVals := []string{e.RouterID["router_id"], rec["name"], rec["remote_address"], rec["local_interface"]}

		metricMap := map[string]struct {
			name       string
			help       string
			parseFloat bool
		}{
			"multiplier":           {"bfd_multiplier", "BFD detection multiplier", true},
			"min_tx_interval":      {"bfd_min_tx_interval", "BFD minimum transmit interval (ms)", true},
			"min_rx_interval":      {"bfd_min_rx_interval", "BFD minimum receive interval (ms)", true},
			"remote-system-id":     {"bfd_remote_system_id", "BFD remote system ID", false},
			"local-discriminator":  {"bfd_local_discriminator", "BFD local discriminator", true},
			"remote-discriminator": {"bfd_remote_discriminator", "BFD remote discriminator", true},
		}

		for key, meta := range metricMap {
			if val, ok := rec[key]; ok && val != "" {
				var value float64
				if meta.parseFloat {
					if key == "multiplier" {
						if v, err := strconv.ParseFloat(val, 64); err == nil {
							value = v
						}
					} else {
						value = ParseFloat(val)
					}
				} else {
					value = 1
				}
				mb.GaugeVal(ch, meta.name, meta.help, value, labelKeysWithRouter, labelVals)
			}
		}

		if state, ok := rec["state"]; ok {
			sessionStatus := 0.0
			if state == "active" || state == "up" {
				sessionStatus = 1
			}
			mb.GaugeVal(ch, "bfd_session_status", "BFD session status (1=up, 0=down)", sessionStatus, labelKeysWithRouter, labelVals)
		}

		if echoMode, ok := rec["echo_mode"]; ok {
			echoEnabled := 0.0
			if strings.ToLower(echoMode) == "yes" || echoMode == trueStr {
				echoEnabled = 1
			}
			mb.GaugeVal(ch, "bfd_echo_mode", "BFD echo mode enabled", echoEnabled, labelKeysWithRouter, labelVals)
		}

		if _, ok := rec["disabled"]; ok {
			disabled := 0.0
			if rec["disabled"] == trueStr {
				disabled = 1
			}
			mb.GaugeVal(ch, "bfd_disabled", "BFD session disabled status", disabled, labelKeysWithRouter, labelVals)
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "bfd_session_info", "Information about BFD session",
				[]string{"name", "remote_address", "local_interface", "comment"},
				rec)
		}

		stateKey := "state"
		if state, ok := rec[stateKey]; ok && state != "" {
			mb.GaugeVal(ch, "bfd_state", "BFD state", ParseFloat(state), labelKeysWithRouter, labelVals)
		}
	}

	return nil
}
