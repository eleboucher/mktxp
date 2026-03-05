package collector

import (
	"context"
	"log/slog"
	"strings"

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

	// Collect IPSec peers (connections)
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
		labelKeysWithRouter := append([]string{"router_id"}, labelKeys...)

		mb.GaugeVal(ch, "ipsec_peer_status", "IPSec peer status (1=active, 0=inactive)", func() float64 {
			if rec["state"] == "running" || rec["state"] == "active" {
				return 1
			}
			return 0
		}(), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})

		if _, ok := rec["disabled"]; ok {
			disabled := 0.0
			if rec["disabled"] == "true" {
				disabled = 1
			}
			mb.GaugeVal(ch, "ipsec_peer_disabled", "IPSec peer disabled status", disabled, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
		}

		if _, ok := rec["auth_algorithm"]; ok && rec["auth_algorithm"] != "" {
			mb.GaugeVal(ch, "ipsec_auth_algorithm", "IPSec authentication algorithm", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
		}

		if _, ok := rec["encryption_algorithm"]; ok && rec["encryption_algorithm"] != "" {
			mb.GaugeVal(ch, "ipsec_encryption_algorithm", "IPSec encryption algorithm", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
		}

		if _, ok := rec["pfs_group"]; ok && rec["pfs_group"] != "" {
			mb.GaugeVal(ch, "ipsec_pfs_group", "IPSec PFS group", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "ipsec_peer_info", "Information about IPSec peer",
				[]string{"name", "remote_address", "auth_algorithm", "encryption_algorithm"},
				rec)
		}

		for _, metric := range []struct{ name, key string }{
			{"ipsec_remote_port", "remote-port"},
			{"ipsec_local_port", "local-port"},
			{"ipsec_lifetime", "lifetime"},
			{"ipsec_pfs", "pfs"},
		} {
			if val, ok := rec[metric.key]; ok && val != "" {
				mb.GaugeVal(ch, metric.name, "IPSec "+strings.ToUpper(metric.key), ParseFloat(val), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
			}
		}
	}

	// Collect IPSec proposals
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

		labelKeysWithRouter := append([]string{"router_id"}, labelKeys...)

		mb.GaugeVal(ch, "ipsec_proposal_enabled", "IPSec proposal enabled status", func() float64 {
			if rec["disabled"] != "true" {
				return 1
			}
			return 0
		}(), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], ""})

		for _, metric := range []struct{ name, key string }{
			{"ipsec_proposal_encryption", "encryption-algorithm"},
			{"ipsec_proposal_authentication", "authentication-algorithms"},
			{"ipsec_proposal_pfs", "pfs-group"},
			{"ipsec_proposal_lifetime", "lifetime"},
		} {
			if val, ok := rec[metric.key]; ok && val != "" {
				mb.GaugeVal(ch, metric.name, "IPSec Proposal "+strings.ToUpper(metric.key), 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], ""})
			}
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "ipsec_proposal_info", "Information about IPSec proposal",
				[]string{"name", "encryption_algorithm", "authentication_algorithms"},
				rec)
		}
	}

	// Collect IPSec policies
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

		labelKeysWithRouter := append([]string{"router_id"}, labelKeys...)

		mb.GaugeVal(ch, "ipsec_policy_enabled", "IPSec policy enabled status", func() float64 {
			if rec["disabled"] != "true" {
				return 1
			}
			return 0
		}(), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], ""})

		mb.GaugeVal(ch, "ipsec_policy_sa_limit", "IPSec policy SA limit", ParseFloat(rec["sa-limit"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], ""})

		for _, metric := range []struct{ name, key string }{
			{"ipsec_policy_tunnel", "tunnel"},
			{"ipsec_policy_src_address", "src-address"},
			{"ipsec_policy_dst_address", "dst-address"},
			{"ipsec_policy_protocol", "protocol"},
			{"ipsec_policy_dst_port", "dst-port"},
			{"ipsec_policy_action", "action"},
		} {
			if val, ok := rec[metric.key]; ok && val != "" {
				mb.GaugeVal(ch, metric.name, "IPSec Policy "+strings.ToUpper(metric.key), 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], ""})
			}
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "ipsec_policy_info", "Information about IPSec policy",
				[]string{"name", "src_address", "dst_address", "action"},
				rec)
		}
	}

	return nil
}
