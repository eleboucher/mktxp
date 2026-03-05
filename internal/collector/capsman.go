package collector

import (
	"context"
	"log/slog"
	"strings"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type CAPsMANCollector struct{}

func NewCAPsMANCollector() *CAPsMANCollector { return &CAPsMANCollector{} }

func (c *CAPsMANCollector) Name() string { return "capsman" }

func (c *CAPsMANCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *CAPsMANCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.CAPsMAN {
		return nil
	}

	mb := NewMetricBuilder(e)

	remoteCaps, err := e.APIConn.Run(ctx, "/caps-man/remote-cap/print")
	if err != nil {
		slog.Debug("capsman remote-cap collect failed", "router", e.RouterName, "err", err)
	} else {
		c.collectRemoteCaps(ctx, e, mb, ch, remoteCaps)
	}

	registrations, err := e.APIConn.Run(ctx, "/caps-man/registration-table/print")
	if err != nil {
		slog.Debug("capsman registration-table collect failed", "router", e.RouterName, "err", err)
	} else {
		c.collectRegistrations(ctx, e, mb, ch, registrations)
	}

	interfaces, err := e.APIConn.Run(ctx, "/caps-man/interface/print")
	if err != nil {
		slog.Debug("capsman interface collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	c.collectInterfaces(ctx, e, mb, ch, interfaces)

	return nil
}

func (c *CAPsMANCollector) collectRemoteCaps(ctx context.Context, e *entry.RouterEntry, mb *MetricBuilder, ch chan<- prometheus.Metric, records []map[string]string) {
	for _, raw := range records {
		rec := TrimRecord(raw, nil)
		if rec["name"] == "" {
			continue
		}

		mb.Info(ch, "capsman_remote_caps", "CAPsMAN remote caps", []string{"router_id", "name", "version", "base_mac", "board"}, rec)
	}
}

func (c *CAPsMANCollector) collectRegistrations(ctx context.Context, e *entry.RouterEntry, mb *MetricBuilder, ch chan<- prometheus.Metric, records []map[string]string) {
	if !e.ConfigEntry.CAPsMANClients {
		return
	}

	interfaceCount := make(map[string]float64)
	var registrationRecords []map[string]string

	for _, raw := range records {
		rec := TrimRecord(raw, nil)
		interfaceName := rec["interface"]
		interfaceCount[interfaceName]++

		registrationRecords = append(registrationRecords, rec)
	}

	for iface, count := range interfaceCount {
		mb.GaugeVal(ch, "capsman_registrations_count", "Number of active registration per CAPsMAN interface",
			count,
			[]string{"router_id", "interface"},
			[]string{e.RouterID["router_id"], iface},
		)
	}

	for _, rec := range registrationRecords {
		rec["dhcp_name"] = rec["host-name"]
		rec["dhcp_address"] = rec["address"]

		txBytes := ParseFloat(rec["tx-bytes"])
		rxBytes := ParseFloat(rec["rx-bytes"])
		signalStrength := ParseFloat(rec["rx-signal"])

		clientLabels := []string{"router_id", "dhcp_name", "mac_address"}
		clientVals := []string{e.RouterID["router_id"], rec["dhcp_name"], rec["mac-address"]}

		mb.GaugeVal(ch, "capsman_clients_tx_bytes", "Number of sent packet bytes", txBytes, clientLabels, clientVals)
		mb.GaugeVal(ch, "capsman_clients_rx_bytes", "Number of received packet bytes", rxBytes, clientLabels, clientVals)
		mb.GaugeVal(ch, "capsman_clients_signal_strength", "Client devices signal strength", signalStrength, clientLabels, clientVals)

		clientInfoLabels := []string{"router_id", "dhcp_name", "dhcp_address", "rx_signal", "ssid", "tx_rate", "rx_rate", "interface", "mac_address", "uptime"}
		mb.Info(ch, "capsman_clients_devices", "Registered client devices info", clientInfoLabels, rec)
	}
}

func (c *CAPsMANCollector) collectInterfaces(ctx context.Context, e *entry.RouterEntry, mb *MetricBuilder, ch chan<- prometheus.Metric, records []map[string]string) {
	for _, raw := range records {
		rec := TrimRecord(raw, nil)
		if rec["name"] == "" {
			continue
		}

		rec["name"] = FormatInterfaceName(rec["name"], "", e.ConfigEntry.InterfaceNameFormat)
		labelKeysWithRouter := append([]string{"router_id"}, "name", "mac_address")
		labelVals := []string{e.RouterID["router_id"], rec["name"], rec["mac-address"]}

		metricMap := map[string]struct {
			name       string
			help       string
			parseFloat bool
		}{
			"mac-address":                {"capsman_mac_address", "CAPsMAN MAC address", false},
			"ip-address":                 {"capsman_ip_address", "CAPsMAN IP address", false},
			"remote-interface":           {"capsman_remote_interface", "CAPsMAN remote interface", false},
			"channel-width":              {"capsman_channel_width", "CAPsMAN channel width", false},
			"current-state":              {"capsman_current_state", "CAPsMAN current state", false},
			"current-channel":            {"capsman_current_channel", "CAPsMAN current channel", false},
			"current-registered-clients": {"capsman_current_registered_clients", "CAPsMAN current registered clients", true},
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
			capsmanStatus := 0.0
			if status == trueStr && rec["disabled"] != trueStr {
				capsmanStatus = 1
			}
			mb.GaugeVal(ch, "capsman_interface_status", "CAPsMAN interface status (1=running, 0=stopped)", capsmanStatus, labelKeysWithRouter, labelVals)
		}

		if _, ok := rec["disabled"]; ok {
			disabled := 0.0
			if rec["disabled"] == trueStr {
				disabled = 1
			}
			mb.GaugeVal(ch, "capsman_interface_disabled", "CAPsMAN interface disabled status", disabled, labelKeysWithRouter, labelVals)
		}

		if _, ok := rec["enabled"]; ok {
			enabled := ParseBool(rec["enabled"])
			mb.GaugeVal(ch, "capsman_interface_enabled", "CAPsMAN interface enabled status", enabled, labelKeysWithRouter, labelVals)
		}

		infoFields := map[string]string{
			"interface-mode": "CAPsMAN interface mode (bridge/access-point/etc)",
			"channel":        "CAPsMAN channel configuration",
			"configuration":  "CAPsMAN configuration",
		}

		for key, help := range infoFields {
			if _, ok := rec[key]; ok && rec[key] != "" {
				mb.GaugeVal(ch, "capsman_"+strings.ReplaceAll(key, "-", "_"), help, 1, labelKeysWithRouter, labelVals)
			}
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "capsman_interfaces", "CAPsMAN interfaces", labelKeysWithRouter, rec)
		}
	}
}
