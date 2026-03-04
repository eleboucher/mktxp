package collector

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

// ratesMbps maps RouterOS rate strings to numeric Mbps values.
var ratesMbps = map[string]float64{
	"10Mbps":  10,
	"100Mbps": 100,
	"1Gbps":   1000,
	"2.5Gbps": 2500,
	"5Gbps":   5000,
	"10Gbps":  10000,
	"40Gbps":  40000,
	"100Gbps": 100000,
}

type MonitorCollector struct{}

func NewMonitorCollector() *MonitorCollector                   { return &MonitorCollector{} }
func (c *MonitorCollector) Name() string                       { return "monitor" }
func (c *MonitorCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *MonitorCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.Monitor {
		return nil
	}

	// Fetch all ethernet interfaces.
	ifaces, err := e.APIConn.Run(ctx, "/interface/ethernet/print", "=.proplist=name,comment,running")
	if err != nil {
		slog.Error("monitor: list interfaces failed", "router", e.RouterName, "err", err)
		return nil
	}

	mb := NewMetricBuilder(e)

	for i, iface := range ifaces {
		name := iface["name"]
		comment := iface["comment"]
		displayName := FormatInterfaceName(name, comment, e.ConfigEntry.InterfaceNameFormat)

		// Only call monitor on running interfaces; for others emit status=0.
		if iface["running"] != "true" {
			mb.GaugeVal(ch, "interface_status", "Current interface link status", 0,
				[]string{"name"}, []string{displayName})
			continue
		}

		mon, err := e.APIConn.Run(ctx, "/interface/ethernet/monitor",
			"=once=",
			"=numbers="+strconv.Itoa(i),
			"=.proplist=status,rate,full-duplex,sfp-module-present,sfp-connector-type,sfp-manufacturing-date,sfp-type,sfp-vendor-name,sfp-vendor-part-number,sfp-vendor-revision,sfp-vendor-serial,sfp-wavelength,sfp-supply-voltage,sfp-rx-power,sfp-tx-power,sfp-temperature,sfp-tx-bias-current,sfp-rx-loss,sfp-tx-fault",
		)
		if err != nil || len(mon) == 0 {
			mb.GaugeVal(ch, "interface_status", "Current interface link status", 0,
				[]string{"name"}, []string{displayName})
			continue
		}

		rec := TrimRecord(mon[0], nil)
		rec["name"] = displayName

		// Status: 1 if link-ok
		status := 0.0
		if rec["status"] == "link-ok" {
			status = 1
		}
		mb.GaugeVal(ch, "interface_status", "Current interface link status", status,
			[]string{"name"}, []string{displayName})

		// Rate in Mbps
		if raw := rec["rate"]; raw != "" {
			if v, ok := ratesMbps[raw]; ok {
				mb.GaugeVal(ch, "interface_rate", "Actual interface connection data rate", v,
					[]string{"name"}, []string{displayName})
			} else if v := ParseFloat(raw); v > 0 {
				mb.GaugeVal(ch, "interface_rate", "Actual interface connection data rate", v,
					[]string{"name"}, []string{displayName})
			}
		}

		// Full duplex
		if raw := rec["full_duplex"]; raw != "" {
			mb.GaugeVal(ch, "interface_full_duplex", "Full duplex data transmission",
				ParseBool(raw), []string{"name"}, []string{displayName})
		}

		// SFP present
		if rec["sfp_module_present"] != "true" {
			continue
		}

		mb.Info(ch, "interface_sfp", "Information about the used transceiver",
			[]string{
				"name", "sfp_connector_type", "sfp_manufacturing_date", "sfp_type",
				"sfp_vendor_name", "sfp_vendor_part_number", "sfp_vendor_revision", "sfp_vendor_serial",
			},
			rec)

		mb.GaugeVal(ch, "interface_sfp_rx_loss", "The receiver signal is lost",
			ParseBool(rec["sfp_rx_loss"]), []string{"name"}, []string{displayName})
		mb.GaugeVal(ch, "interface_sfp_tx_fault", "The transceiver transmitter is in fault state",
			ParseBool(rec["sfp_tx_fault"]), []string{"name"}, []string{displayName})

		for _, dom := range []struct{ metric, help, key string }{
			{"interface_sfp_supply_voltage", "Transceiver supply voltage", "sfp_supply_voltage"},
			{"interface_sfp_rx_power", "Current SFP RX power", "sfp_rx_power"},
			{"interface_sfp_tx_power", "Current SFP TX power", "sfp_tx_power"},
			{"interface_sfp_temperature", "Current SFP temperature", "sfp_temperature"},
			{"interface_sfp_wavelength", "Current SFP wavelength", "sfp_wavelength"},
		} {
			if v := ParseFloat(rec[dom.key]); rec[dom.key] != "" {
				mb.GaugeVal(ch, dom.metric, dom.help, v, []string{"name"}, []string{displayName})
			}
		}

		// TX bias current is reported in µA, convert to mA
		if raw := rec["sfp_tx_bias_current"]; raw != "" {
			if v := ParseFloat(raw); v != 0 {
				mb.GaugeVal(ch, "interface_sfp_tx_bias_current", "Transceiver TX bias current (mA)",
					v/1000, []string{"name"}, []string{displayName})
			}
		}
	}

	return nil
}
