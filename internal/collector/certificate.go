package collector

import (
	"context"
	"log/slog"
	"strings"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type CertificateCollector struct{}

func NewCertificateCollector() *CertificateCollector { return &CertificateCollector{} }

func (c *CertificateCollector) Name() string { return "certificate" }

func (c *CertificateCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *CertificateCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.Certificate {
		return nil
	}

	records, err := e.APIConn.Run(ctx, "/certificate/print")
	if err != nil {
		slog.Debug("certificate collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	mb := NewMetricBuilder(e)
	labelKeys := []string{"name", "common_name", "issuer"}

	for _, raw := range records {
		rec := TrimRecord(raw, nil)
		if rec["name"] == "" {
			continue
		}

		rec["name"] = FormatInterfaceName(rec["name"], "", e.ConfigEntry.InterfaceNameFormat)
		labelKeysWithRouter := append([]string{"router_id"}, labelKeys...)

		mb.GaugeVal(ch, "certificate_valid", "Certificate validity status (1=valid, 0=expired/invalid)", func() float64 {
			if rec["expires-after"] != "" && rec["disabled"] != "true" {
				return 1
			}
			return 0
		}(), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["common_name"], rec["issuer"]})

		if _, ok := rec["disabled"]; ok {
			disabled := 0.0
			if rec["disabled"] == "true" {
				disabled = 1
			}
			mb.GaugeVal(ch, "certificate_disabled", "Certificate disabled status", disabled, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["common_name"], rec["issuer"]})
		}

		if _, ok := rec["expires-after"]; ok && rec["expires-after"] != "" {
			mb.GaugeVal(ch, "certificate_expires_after", "Certificate expiration time in seconds", ParseFloat(rec["expires-after"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["common_name"], rec["issuer"]})
		}

		if _, ok := rec["key-type"]; ok && rec["key-type"] != "" {
			mb.GaugeVal(ch, "certificate_key_type", "Certificate key type (RSA/ECDSA)", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["common_name"], rec["issuer"]})
		}

		if _, ok := rec["key-size"]; ok && rec["key-size"] != "" {
			mb.GaugeVal(ch, "certificate_key_size", "Certificate key size in bits", ParseFloat(rec["key-size"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["common_name"], rec["issuer"]})
		}

		if _, ok := rec["serial-number"]; ok && rec["serial-number"] != "" {
			mb.GaugeVal(ch, "certificate_serial_number", "Certificate serial number", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["common_name"], rec["issuer"]})
		}

		if _, ok := rec["fingerprint"]; ok && rec["fingerprint"] != "" {
			mb.GaugeVal(ch, "certificate_fingerprint", "Certificate fingerprint", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["common_name"], rec["issuer"]})
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "certificate_info", "Information about certificate",
				[]string{"name", "common_name", "issuer", "key_type"},
				rec)
		}

		for _, metric := range []struct{ name, key string }{
			{"certificate_issuer_cn", "issuer-cn"},
			{"certificate_not_before", "not-before"},
			{"certificate_not_after", "not-after"},
			{"certificate_common_name", "common-name"},
		} {
			if val, ok := rec[metric.key]; ok && val != "" {
				mb.GaugeVal(ch, metric.name, "Certificate "+strings.ToUpper(metric.key), 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["common_name"], rec["issuer"]})
			}
		}
	}

	return nil
}
