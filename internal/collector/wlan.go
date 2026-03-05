package collector

import (
	"context"
	"log/slog"
	"strings"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type WLANCollector struct{}

func NewWLANCollector() *WLANCollector { return &WLANCollector{} }

func (c *WLANCollector) Name() string { return "wlan" }

func (c *WLANCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *WLANCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.Wireless {
		return nil
	}

	mb := NewMetricBuilder(e)

	monitorRecords, err := e.APIConn.Run(ctx, "/interface/wireless/monitor")
	if err != nil {
		slog.Debug("wlan monitor collect failed", "router", e.RouterName, "err", err)
	} else {
		c.collectMonitor(e, mb, ch, monitorRecords)
	}

	registrationRecords, err := e.APIConn.Run(ctx, "/interface/wireless/registration-table/print")
	if err != nil {
		slog.Debug("wlan registration-table collect failed", "router", e.RouterName, "err", err)
	} else {
		c.collectRegistrations(e, mb, ch, registrationRecords)
	}

	interfaceRecords, err := e.APIConn.Run(ctx, "/interface/wireless/print")
	if err != nil {
		slog.Debug("wlan interface collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	c.collectInterfaces(e, mb, ch, interfaceRecords)

	return nil
}

func (c *WLANCollector) collectMonitor(e *entry.RouterEntry, mb *MetricBuilder, ch chan<- prometheus.Metric, records []map[string]string) {
	var noiseFloorRecords []map[string]string
	var txCCQRecords []map[string]string
	var registeredClientsRecords []map[string]string

	for _, raw := range records {
		rec := TrimRecord(raw, nil)
		if rec["name"] == "" {
			continue
		}

		if rec["noise-floor"] != "" {
			noiseFloorRecords = append(noiseFloorRecords, rec)
		}
		if rec["overall-tx-ccq"] != "" {
			txCCQRecords = append(txCCQRecords, rec)
		}
		if rec["registered-clients"] != "" || rec["registered-peers"] != "" {
			registeredClientsRecords = append(registeredClientsRecords, rec)
		}
	}

	if len(noiseFloorRecords) > 0 {
		metricMap := map[string]struct {
			name       string
			help       string
			parseFloat bool
		}{
			"noise-floor": {"wlan_noise_floor", "Noise floor threshold", true},
		}

		for _, rec := range noiseFloorRecords {
			labelKeys := []string{"channel"}
			labelVals := []string{rec["channel"]}

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
	}

	if len(txCCQRecords) > 0 {
		metricMap := map[string]struct {
			name       string
			help       string
			parseFloat bool
		}{
			"overall-tx-ccq": {"wlan_overall_tx_ccq", "Client Connection Quality for transmitting", true},
		}

		for _, rec := range txCCQRecords {
			labelKeys := []string{"channel"}
			labelVals := []string{rec["channel"]}

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
	}

	if len(registeredClientsRecords) > 0 {
		metricMap := map[string]struct {
			name       string
			help       string
			parseFloat bool
		}{
			"registered-clients": {"wlan_registered_clients", "Number of registered clients", true},
			"registered-peers":   {"wlan_registered_clients", "Number of registered clients", true},
		}

		for _, rec := range registeredClientsRecords {
			labelKeys := []string{"channel"}
			labelVals := []string{rec["channel"]}

			for key, meta := range metricMap {
				if val, ok := rec[key]; ok && val != "" {
					var value float64
					if meta.parseFloat {
						value = ParseFloat(val)
					} else {
						value = 1
					}
					mb.GaugeVal(ch, meta.name, meta.help, value, labelKeys, labelVals)
					break
				}
			}
		}
	}
}

func (c *WLANCollector) collectRegistrations(e *entry.RouterEntry, mb *MetricBuilder, ch chan<- prometheus.Metric, records []map[string]string) {
	if !e.ConfigEntry.WirelessClients {
		return
	}

	var registrationRecords []map[string]string

	for _, raw := range records {
		rec := TrimRecord(raw, nil)
		rec["dhcp_name"] = rec["host-name"]
		rec["dhcp_address"] = rec["address"]
		registrationRecords = append(registrationRecords, rec)
	}

	for _, rec := range registrationRecords {
		txBytes := ParseFloat(rec["tx-bytes"])
		rxBytes := ParseFloat(rec["rx-bytes"])
		signalStrength := ParseFloat(rec["signal-strength"])
		signalToNoise := ParseFloat(rec["signal-to-noise"])
		txCCQ := ParseFloat(rec["tx-ccq"])

		clientLabels := []string{"dhcp_name", "mac_address"}
		clientVals := []string{rec["dhcp_name"], rec["mac-address"]}

		mb.GaugeVal(ch, "wlan_clients_tx_bytes", "Number of sent packet bytes", txBytes, clientLabels, clientVals)
		mb.GaugeVal(ch, "wlan_clients_rx_bytes", "Number of received packet bytes", rxBytes, clientLabels, clientVals)
		mb.GaugeVal(ch, "wlan_clients_signal_strength", "Average strength of the client signal recevied by AP", signalStrength, clientLabels, clientVals)
		mb.GaugeVal(ch, "wlan_clients_signal_to_noise", "Client devices signal to noise ratio", signalToNoise, clientLabels, clientVals)
		mb.GaugeVal(ch, "wlan_clients_tx_ccq", "Client Connection Quality (CCQ) for transmit", txCCQ, clientLabels, clientVals)

		clientInfoLabels := []string{"dhcp_name", "dhcp_address", "rx_signal", "ssid", "tx_rate", "rx_rate", "interface", "mac_address", "uptime"}
		mb.Info(ch, "wlan_clients_devices", "Client devices info", clientInfoLabels, rec)
	}
}

func (c *WLANCollector) collectInterfaces(e *entry.RouterEntry, mb *MetricBuilder, ch chan<- prometheus.Metric, records []map[string]string) {
	for _, raw := range records {
		rec := TrimRecord(raw, nil)
		if rec["name"] == "" {
			continue
		}

		rec["name"] = FormatInterfaceName(rec["name"], "", e.ConfigEntry.InterfaceNameFormat)
		labelKeysWithRouter := []string{"name", "ssid"}
		labelVals := []string{rec["name"], rec["ssid"]}

		metricMap := map[string]struct {
			name       string
			help       string
			parseFloat bool
		}{
			"ssid":             {"wlan_ssid", "WLAN SSID", false},
			"bssid":            {"wlan_bssid", "WLAN BSSID", false},
			"channel":          {"wlan_channel", "WLAN Channel", false},
			"bridge-mode":      {"wlan_bridge_mode", "WLAN Bridge Mode", false},
			"security-profile": {"wlan_security_profile", "WLAN Security Profile", false},
			"channel-width":    {"wlan_channel_width", "WLAN channel width in MHz", true},
			"frequency":        {"wlan_frequency", "WLAN operating frequency in MHz", true},
			"tx-power":         {"wlan_tx_power", "WLAN transmit power in dBm", true},
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
			wlanStatus := 0.0
			if status == trueStr && rec["disabled"] != trueStr {
				wlanStatus = 1
			}
			mb.GaugeVal(ch, "wlan_status", "WLAN interface status (1=up, 0=down)", wlanStatus, labelKeysWithRouter, labelVals)
		}

		if _, ok := rec["disabled"]; ok {
			disabled := 0.0
			if rec["disabled"] == trueStr {
				disabled = 1
			}
			mb.GaugeVal(ch, "wlan_disabled", "WLAN interface disabled status", disabled, labelKeysWithRouter, labelVals)
		}

		infoFields := map[string]string{
			"band":              "WLAN frequency band (2g/5g)",
			"country":           "WLAN regulatory domain country",
			"mode":              "WLAN operating mode (ap-bridge/station/etc)",
			"wireless-protocol": "WLAN protocol (802.11a/b/g/n/ac/ax)",
		}

		for key, help := range infoFields {
			if _, ok := rec[key]; ok && rec[key] != "" {
				mb.GaugeVal(ch, "wlan_"+strings.ReplaceAll(key, "-", "_"), help, 1, labelKeysWithRouter, labelVals)
			}
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "wlan_info", "Information about WLAN interface",
				[]string{"name", "ssid", "band", "mode"},
				rec)
		}
	}
}
