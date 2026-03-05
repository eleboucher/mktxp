package collector

import (
	"context"
	"log/slog"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type IPSecCollector struct{}

func NewIPSecCollector() *IPSecCollector { return &IPSecCollector{} }

func (c *IPSecCollector) Name() string { return "ipsec" }

func (c *IPSecCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *IPSecCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.IPSec {
		return nil
	}

	peers, err := e.APIConn.Run(ctx, "/ip/ipsec/peer/print")
	if err != nil {
		slog.Debug("ipsec peer collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	mb := NewMetricBuilder(e)
	labelKeys := []string{"name", "remote_address"}

	for _, raw := range peers {
		rec := TrimRecord(raw, nil)
		if rec["name"] == "" {
			continue
		}

		rec["name"] = FormatInterfaceName(rec["name"], "", e.ConfigEntry.InterfaceNameFormat)
		labelKeysWithRouter := append([]string{"routerboard_name"}, labelKeys...)

		collectIPSecPeer(mb, ch, rec, labelKeysWithRouter, e.RouterID)
	}

	proposals, err := e.APIConn.Run(ctx, "/ip/ipsec/proposal/print")
	if err != nil {
		slog.Debug("ipsec proposal collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	for _, raw := range proposals {
		rec := TrimRecord(raw, nil)
		if rec["name"] == "" {
			continue
		}

		labelKeysWithRouter := append([]string{"routerboard_name"}, labelKeys...)
		collectIPSecProposal(mb, ch, rec, labelKeysWithRouter, e.RouterID)
	}

	policies, err := e.APIConn.Run(ctx, "/ip/ipsec/policy/print")
	if err != nil {
		slog.Debug("ipsec policy collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	for _, raw := range policies {
		rec := TrimRecord(raw, nil)
		if rec["name"] == "" {
			continue
		}

		labelKeysWithRouter := append([]string{"routerboard_name"}, labelKeys...)
		collectIPSecPolicy(mb, ch, rec, labelKeysWithRouter, e.RouterID)
	}

	return nil
}

func collectIPSecPeer(mb *MetricBuilder, ch chan<- prometheus.Metric, rec map[string]string, labelKeysWithRouter []string, routerID map[string]string) {
	metricMap := map[string]struct {
		name       string
		help       string
		parseFloat bool
	}{
		"remote-port":           {"ipsec_remote_port", "IPSec remote port", true},
		"local-port":            {"ipsec_local_port", "IPSec local port", true},
		"lifetime":              {"ipsec_lifetime", "IPSec lifetime", true},
		"pfs":                   {"ipsec_pfs", "IPSec PFS", true},
		"last-seen":             {"ipsec_peer_last_seen", "IPSec peer last seen timestamp", true},
		"natt-enabled":          {"ipsec_peer_natt_enabled", "IPSec peer NAT-T enabled", false},
		"responder":             {"ipsec_peer_responder", "IPSec peer responder mode", false},
		"rx-bytes":              {"ipsec_peer_rx_byte", "IPSec peer received bytes", true},
		"tx-bytes":              {"ipsec_peer_tx_byte", "IPSec peer transmitted bytes", true},
		"rx-packets":            {"ipsec_peer_rx_packet", "IPSec peer received packets", true},
		"tx-packets":            {"ipsec_peer_tx_packet", "IPSec peer transmitted packets", true},
		"security-associations": {"ipsec_peer_security_association", "IPSec peer security associations", true},
		"state":                 {"ipsec_peer_state", "IPSec peer state", false},
		"uptime":                {"ipsec_peer_uptime", "IPSec peer uptime", true},
	}

	for key, meta := range metricMap {
		if val, ok := rec[key]; ok && val != "" {
			var value float64
			if meta.parseFloat {
				value = ParseFloat(val)
			} else {
				value = 1.0
			}
			mb.GaugeVal(ch, meta.name, meta.help, value, labelKeysWithRouter, []string{routerID["routerboard_name"], rec["name"], rec["remote_address"]})
		}
	}

	if rec["state"] == "running" || rec["state"] == "active" {
		mb.GaugeVal(ch, "ipsec_peer_status", "IPSec peer status (1=active, 0=inactive)", 1.0, labelKeysWithRouter, []string{routerID["routerboard_name"], rec["name"], rec["remote_address"]})
	} else {
		mb.GaugeVal(ch, "ipsec_peer_status", "IPSec peer status (1=active, 0=inactive)", 0.0, labelKeysWithRouter, []string{routerID["routerboard_name"], rec["name"], rec["remote_address"]})
	}

	if disabledVal, ok := rec["disabled"]; ok {
		disabled := 0.0
		if disabledVal == trueStr {
			disabled = 1.0
		}
		mb.GaugeVal(ch, "ipsec_peer_disabled", "IPSec peer disabled status", disabled, labelKeysWithRouter, []string{routerID["routerboard_name"], rec["name"], rec["remote_address"]})
	}

	if _, ok := rec["auth_algorithm"]; ok && rec["auth_algorithm"] != "" {
		mb.GaugeVal(ch, "ipsec_auth_algorithm", "IPSec authentication algorithm", 1.0, labelKeysWithRouter, []string{routerID["routerboard_name"], rec["name"], rec["remote_address"]})
	}

	if _, ok := rec["encryption_algorithm"]; ok && rec["encryption_algorithm"] != "" {
		mb.GaugeVal(ch, "ipsec_encryption_algorithm", "IPSec encryption algorithm", 1.0, labelKeysWithRouter, []string{routerID["routerboard_name"], rec["name"], rec["remote_address"]})
	}

	if _, ok := rec["pfs_group"]; ok && rec["pfs_group"] != "" {
		mb.GaugeVal(ch, "ipsec_pfs_group", "IPSec PFS group", 1.0, labelKeysWithRouter, []string{routerID["routerboard_name"], rec["name"], rec["remote_address"]})
	}

	if comment, ok := rec["comment"]; ok && comment != "" {
		mb.Info(ch, "ipsec_peer_info", "Information about IPSec peer",
			[]string{"name", "remote_address", "auth_algorithm", "encryption_algorithm"},
			rec)
	}
}

func collectIPSecProposal(mb *MetricBuilder, ch chan<- prometheus.Metric, rec map[string]string, labelKeysWithRouter []string, routerID map[string]string) {
	metricMap := map[string]struct {
		name       string
		help       string
		parseFloat bool
	}{
		"encryption-algorithm":      {"ipsec_proposal_encryption", "IPSec proposal encryption", false},
		"authentication-algorithms": {"ipsec_proposal_authentication", "IPSec proposal authentication", false},
		"pfs-group":                 {"ipsec_proposal_pfs", "IPSec proposal PFS", false},
		"lifetime":                  {"ipsec_proposal_lifetime", "IPSec proposal lifetime", true},
	}

	for key, meta := range metricMap {
		if val, ok := rec[key]; ok && val != "" {
			var value float64
			if meta.parseFloat {
				value = ParseFloat(val)
			} else {
				value = 1.0
			}
			mb.GaugeVal(ch, meta.name, meta.help, value, labelKeysWithRouter, []string{routerID["routerboard_name"], rec["name"], ""})
		}
	}

	if rec["disabled"] != trueStr {
		mb.GaugeVal(ch, "ipsec_proposal_enabled", "IPSec proposal enabled status", 1.0, labelKeysWithRouter, []string{routerID["routerboard_name"], rec["name"], ""})
	} else {
		mb.GaugeVal(ch, "ipsec_proposal_enabled", "IPSec proposal enabled status", 0.0, labelKeysWithRouter, []string{routerID["routerboard_name"], rec["name"], ""})
	}

	if comment, ok := rec["comment"]; ok && comment != "" {
		mb.Info(ch, "ipsec_proposal_info", "Information about IPSec proposal",
			[]string{"name", "encryption_algorithm", "authentication_algorithms"},
			rec)
	}
}

func collectIPSecPolicy(mb *MetricBuilder, ch chan<- prometheus.Metric, rec map[string]string, labelKeysWithRouter []string, routerID map[string]string) {
	metricMap := map[string]struct {
		name       string
		help       string
		parseFloat bool
	}{
		"tunnel":      {"ipsec_policy_tunnel", "IPSec policy tunnel", false},
		"src-address": {"ipsec_policy_src_address", "IPSec policy source address", false},
		"dst-address": {"ipsec_policy_dst_address", "IPSec policy destination address", false},
		"protocol":    {"ipsec_policy_protocol", "IPSec policy protocol", false},
		"dst-port":    {"ipsec_policy_dst_port", "IPSec policy destination port", true},
		"action":      {"ipsec_policy_action", "IPSec policy action", false},
		"sa-limit":    {"ipsec_policy_sa_limit", "IPSec policy SA limit", true},
	}

	for key, meta := range metricMap {
		if val, ok := rec[key]; ok && val != "" {
			var value float64
			if meta.parseFloat {
				value = ParseFloat(val)
			} else {
				value = 1.0
			}
			mb.GaugeVal(ch, meta.name, meta.help, value, labelKeysWithRouter, []string{routerID["routerboard_name"], rec["name"], ""})
		}
	}

	if rec["disabled"] != trueStr {
		mb.GaugeVal(ch, "ipsec_policy_enabled", "IPSec policy enabled status", 1.0, labelKeysWithRouter, []string{routerID["routerboard_name"], rec["name"], ""})
	} else {
		mb.GaugeVal(ch, "ipsec_policy_enabled", "IPSec policy enabled status", 0.0, labelKeysWithRouter, []string{routerID["routerboard_name"], rec["name"], ""})
	}

	if comment, ok := rec["comment"]; ok && comment != "" {
		mb.Info(ch, "ipsec_policy_info", "Information about IPSec policy",
			[]string{"name", "src_address", "dst_address", "action"},
			rec)
	}
}
