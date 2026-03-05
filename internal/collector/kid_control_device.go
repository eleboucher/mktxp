package collector

import (
	"context"
	"log/slog"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/eleboucher/mktxp/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

type KidControlDeviceCollector struct{}

func NewKidControlDeviceCollector() *KidControlDeviceCollector {
	return &KidControlDeviceCollector{}
}

func (c *KidControlDeviceCollector) Name() string { return "kid_control_device" }

func (c *KidControlDeviceCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *KidControlDeviceCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.KidControlAssigned && !e.ConfigEntry.KidControlDynamic {
		return nil
	}

	records, err := e.APIConn.Run(ctx, "/interface/wireless/kid-control/registration/print")
	if err != nil {
		slog.Debug("kid_control_device collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	mb := NewMetricBuilder(e)
	labelKeys := []string{"name", "user", "mac_address", "ip_address"}

	for _, raw := range records {
		rec := TrimRecord(raw, nil)
		if rec["name"] == "" {
			continue
		}

		rec["name"] = FormatInterfaceName(rec["name"], "", e.ConfigEntry.InterfaceNameFormat)
		labelKeysWithRouter := labelKeys
		labelVals := []string{rec["name"], rec["mac_address"], rec["ip_address"]}

		metricMap := map[string]struct {
			name       string
			help       string
			parseFloat bool
		}{
			"bytes-down": {"kid_control_device_bytes_down", "Number of received bytes", true},
			"bytes-up":   {"kid_control_device_bytes_up", "Number of transmitted bytes", true},
			"rate-up":    {"kid_control_device_rate_up", "Device rate up", true},
			"rate-down":  {"kid_control_device_rate_down", "Device rate down", true},
			"idle-time":  {"kid_control_device_idle_time", "Device idle time", true},
		}

		for key, meta := range metricMap {
			if val, ok := rec[key]; ok && val != "" {
				var value float64
				if meta.parseFloat {
					if key == "idle-time" {
						value = utils.ParseTimedelta(val, true)
					} else {
						value = ParseFloat(val)
					}
				} else {
					value = 1.0
				}
				mb.GaugeVal(ch, meta.name, meta.help, value, labelKeysWithRouter, labelVals)
			}
		}

		if blockedVal, ok := rec["blocked"]; ok {
			blocked := 0.0
			if blockedVal == trueStr {
				blocked = 1.0
			}
			mb.GaugeVal(ch, "kid_control_blocked", "Kid Control blocked status", blocked, labelKeysWithRouter, labelVals)
		}

		if limitedVal, ok := rec["limited"]; ok {
			limited := 0.0
			if limitedVal == trueStr {
				limited = 1.0
			}
			mb.GaugeVal(ch, "kid_control_limited", "Kid Control limited status", limited, labelKeysWithRouter, labelVals)
		}

		if inactiveVal, ok := rec["inactive"]; ok {
			inactive := 0.0
			if inactiveVal == trueStr {
				inactive = 1.0
			}
			mb.GaugeVal(ch, "kid_control_inactive", "Kid Control inactive status", inactive, labelKeysWithRouter, labelVals)
		}

		if disabledVal, ok := rec["disabled"]; ok {
			disabled := 0.0
			if disabledVal == trueStr {
				disabled = 1.0
			}
			mb.GaugeVal(ch, "kid_control_disabled", "Kid Control disabled status", disabled, labelKeysWithRouter, labelVals)
		}

		infoLabels := labelKeys
		mb.Info(ch, "kid_control_device", "Kid-control device Info", infoLabels, rec)
	}

	return nil
}
