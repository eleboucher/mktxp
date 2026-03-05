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
	labelKeysWithRouter := []string{"routerboard_name", "name", "common_name", "issuer"}

	for _, raw := range records {
		rec := TrimRecord(raw, nil)
		if rec["name"] == "" {
			continue
		}

		rec["name"] = FormatInterfaceName(rec["name"], "", e.ConfigEntry.InterfaceNameFormat)
		labelVals := []string{e.RouterID["routerboard_name"], rec["name"], rec["common_name"], rec["issuer"]}

		metricMap := map[string]struct {
			name       string
			help       string
			parseFloat bool
		}{
			"expires-after": {"certificate_expires_after", "Certificate expiration time in seconds", true},
			"key-size":      {"certificate_key_size", "Certificate key size in bits", true},
		}

		for key, meta := range metricMap {
			if val, ok := rec[key]; ok && val != "" {
				var value float64
				if meta.parseFloat {
					value = ParseFloat(val)
				} else {
					value = 1.0
				}
				mb.GaugeVal(ch, meta.name, meta.help, value, labelKeysWithRouter, labelVals)
			}
		}

		if _, ok := rec["disabled"]; ok {
			disabled := 0.0
			if rec["disabled"] == trueStr {
				disabled = 1
			}
			mb.GaugeVal(ch, "certificate_disabled", "Certificate disabled status", disabled, labelKeysWithRouter, labelVals)
		}

		if rec["expires-after"] != "" && rec["disabled"] != trueStr {
			mb.GaugeVal(ch, "certificate_valid", "Certificate validity status (1=valid, 0=expired/invalid)", 1, labelKeysWithRouter, labelVals)
		} else {
			mb.GaugeVal(ch, "certificate_valid", "Certificate validity status (1=valid, 0=expired/invalid)", 0, labelKeysWithRouter, labelVals)
		}

		metricInfo := map[string]string{
			"key-type":      "Certificate key type (RSA/ECDSA)",
			"serial-number": "Certificate serial number",
			"fingerprint":   "Certificate fingerprint",
		}

		for key, help := range metricInfo {
			if _, ok := rec[key]; ok && rec[key] != "" {
				mb.GaugeVal(ch, "certificate_"+strings.ReplaceAll(key, "-", "_"), help, 1, labelKeysWithRouter, labelVals)
			}
		}

		metricFields := []struct{ metric, key string }{
			{"certificate_issuer_cn", "issuer-cn"},
			{"certificate_not_before", "not-before"},
			{"certificate_not_after", "not-after"},
			{"certificate_common_name", "common-name"},
		}

		for _, mf := range metricFields {
			if val, ok := rec[mf.key]; ok && val != "" {
				mb.GaugeVal(ch, mf.metric, "Certificate "+strings.ToUpper(mf.key), 1, labelKeysWithRouter, labelVals)
			}
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "certificate_expiration_timestamp_seconds", "Information about certificate",
				[]string{"name", "common_name", "issuer", "key_type"}, rec)
		}
	}

	return nil
}
