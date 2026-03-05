package collector

import (
	"context"
	"fmt"
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

	ifaces, err := e.APIConn.Run(ctx, "/interface/ethernet/print", "=.proplist=name,comment,running")
	if err != nil {
		slog.Error("monitor: list interfaces failed", "router", e.RouterName, "err", err)
		return fmt.Errorf("monitor: %w", err)
	}

	mb := NewMetricBuilder(e)

	for i, iface := range ifaces {
		name := iface["name"]
		comment := iface["comment"]
		displayName := FormatInterfaceName(name, comment, e.ConfigEntry.InterfaceNameFormat)

		if iface["running"] != trueStr {
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

		status := 0.0
		if rec["status"] == "link-ok" {
			status = 1
		}
		mb.GaugeVal(ch, "interface_status", "Current interface link status", status,
			[]string{"name"}, []string{displayName})

		labelKeys := []string{"name"}
		labelVals := []string{displayName}

		metricMap := map[string]struct {
			name       string
			help       string
			parseFloat bool
		}{
			"rate":                {"interface_rate", "Actual interface connection data rate", false},
			"full_duplex":         {"interface_full_duplex", "Full duplex data transmission", false},
			"sfp_supply_voltage":  {"interface_sfp_supply_voltage", "Transceiver supply voltage", true},
			"sfp_rx_power":        {"interface_sfp_rx_power", "Current SFP RX power", true},
			"sfp_tx_power":        {"interface_sfp_tx_power", "Current SFP TX power", true},
			"sfp_temperature":     {"interface_sfp_temperature", "Current SFP temperature", true},
			"sfp_wavelength":      {"interface_sfp_wavelength", "Current SFP wavelength", true},
			"sfp_tx_bias_current": {"interface_sfp_tx_bias_current", "Transceiver TX bias current (mA)", true},
		}

		for key, meta := range metricMap {
			if val, ok := rec[key]; ok && val != "" {
				var value float64
				if meta.parseFloat {
					switch key {
					case "rate":
						if v, ok := ratesMbps[val]; ok {
							value = v
						} else {
							value = ParseFloat(val)
						}
					case "sfp_tx_bias_current":
						value = ParseFloat(val) / 1000
					default:
						value = ParseFloat(val)
					}
				} else {
					value = ParseBool(val)
				}
				mb.GaugeVal(ch, meta.name, meta.help, value, labelKeys, labelVals)
			}
		}

		if rec["sfp_module_present"] != trueStr {
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
	}

	return nil
}
