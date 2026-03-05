package collector

import (
	"context"
	"log/slog"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type AddressListCollector struct{}

func NewAddressListCollector() *AddressListCollector { return &AddressListCollector{} }

func (c *AddressListCollector) Name() string { return "address_list" }

func (c *AddressListCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *AddressListCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if len(e.ConfigEntry.AddressList) == 0 {
		return nil
	}

	records, err := e.APIConn.Run(ctx, "/ip/address/print")
	if err != nil {
		slog.Debug("address_list collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	mb := NewMetricBuilder(e)
	labelKeys := []string{"interface", "address", "network"}

	for _, raw := range records {
		rec := TrimRecord(raw, nil)
		if rec["interface"] == "" {
			continue
		}

		rec["name"] = FormatInterfaceName(rec["interface"], "", e.ConfigEntry.InterfaceNameFormat)
		labelKeysWithRouter := append([]string{"router_id"}, labelKeys...)
		labelVals := []string{e.RouterID["router_id"], rec["name"], rec["address"], rec["network"]}

		metricMap := map[string]struct {
			name       string
			help       string
			parseFloat bool
		}{
			"address":  {"ip_address_assigned", "Indicates if an IP address is assigned to an interface", false},
			"dynamic":  {"ip_address_dynamic", "Indicates if the IP address is dynamically assigned", false},
			"disabled": {"ip_address_disabled", "Indicates if the IP address is disabled", false},
		}

		for key, meta := range metricMap {
			if val, ok := rec[key]; ok && val != "" {
				var value float64
				if meta.parseFloat {
					value = ParseFloat(val)
				} else {
					if key == "address" {
						value = 1
					} else {
						value = ParseBool(val)
					}
				}
				mb.GaugeVal(ch, meta.name, meta.help, value, labelKeysWithRouter, labelVals)
			}
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "ip_address_info", "Information about IP address assignment",
				[]string{"interface", "address", "network", "comment"},
				rec)
		}
	}

	return nil
}
