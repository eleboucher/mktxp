package collector

import (
	"context"
	"log/slog"
	"time"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type UserCollector struct{}

func NewUserCollector() *UserCollector                      { return &UserCollector{} }
func (c *UserCollector) Name() string                       { return "user" }
func (c *UserCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *UserCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.User {
		return nil
	}

	records, err := e.APIConn.Run(ctx, "/user/active/print", "=.proplist=.id,name,when,address,via,group")
	if err != nil {
		slog.Error("user collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	mb := NewMetricBuilder(e)
	labelKeys := []string{"session_id", "name", "address", "via", "group"}

	for _, raw := range records {
		rec := TrimRecord(raw, append(labelKeys, "when"))

		// Parse login timestamp; fall back to now on failure.
		ts := float64(time.Now().Unix())
		if raw["when"] != "" {
			if t, err := time.ParseInLocation("2006-01-02 15:04:05", raw["when"], time.Local); err == nil {
				ts = float64(t.Unix())
			}
		}

		labelVals := make([]string, len(labelKeys))
		for i, k := range labelKeys {
			labelVals[i] = rec[k]
		}
		mb.GaugeVal(ch, "active_users_info", "Active users login timestamp", ts, labelKeys, labelVals)
	}

	return nil
}
