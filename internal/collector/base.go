package collector

import (
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/cespare/xxhash/v2"
	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/eleboucher/mktxp/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "mktxp"

const (
	trueStr = "true"
)

var keyCache sync.Map

// NormalizeKey replaces RouterOS key separators (. and -) with underscores.
func NormalizeKey(key string) string {
	if cached, ok := keyCache.Load(key); ok {
		return cached.(string)
	}

	// If not, do the expensive allocation once
	normalized := strings.ReplaceAll(key, ".", "_")
	normalized = strings.ReplaceAll(normalized, "-", "_")

	keyCache.Store(key, normalized)
	return normalized
}

func ParseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func ParseBool(s string) float64 {
	switch strings.ToLower(s) {
	case trueStr, "yes":
		return 1
	default:
		return 0
	}
}

// TrimRecord filters a RouterOS API record to the requested keys with normalized names.
// If wantedKeys is nil/empty, all keys are returned (normalized).
func TrimRecord(record map[string]string, wantedKeys []string) map[string]string {
	if len(wantedKeys) == 0 {
		out := make(map[string]string, len(record))
		for k, v := range record {
			out[NormalizeKey(k)] = v
		}
		return out
	}
	wanted := make(map[string]struct{}, len(wantedKeys))
	for _, k := range wantedKeys {
		wanted[NormalizeKey(k)] = struct{}{}
	}
	out := make(map[string]string, len(wantedKeys))
	for k, v := range record {
		nk := NormalizeKey(k)
		if _, ok := wanted[nk]; ok {
			out[nk] = v
		}
	}
	return out
}

// FormatInterfaceName delegates to utils.FormatInterfaceName.
func FormatInterfaceName(name, comment, mode string) string {
	return utils.FormatInterfaceName(name, comment, mode)
}

// MetricBuilder emits Prometheus metrics with consistent router-ID and custom labels
// appended to every metric. Uses the unchecked collector pattern (Describe is always a no-op).
type MetricBuilder struct {
	routerID     map[string]string
	customKeys   []string // sorted for deterministic label ordering
	customLabels map[string]string

	emitted map[string]map[uint64]struct{}
}

func NewMetricBuilder(e *entry.RouterEntry) *MetricBuilder {
	cl := e.ConfigEntry.CustomLabels
	keys := make([]string, 0, len(cl))
	for k := range cl {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return &MetricBuilder{
		routerID:     e.RouterID,
		customKeys:   keys,
		customLabels: cl,
		emitted:      make(map[string]map[uint64]struct{}),
	}
}

func (b *MetricBuilder) labelNames(extra []string) []string {
	out := make([]string, 0, len(extra)+2+len(b.customKeys))
	out = append(out, extra...)
	out = append(out, "routerboard_name", "routerboard_address")
	out = append(out, b.customKeys...)
	return out
}

func (b *MetricBuilder) labelVals(extra []string) []string {
	out := make([]string, 0, len(extra)+2+len(b.customKeys))
	out = append(out, extra...)
	out = append(out, b.routerID["routerboard_name"], b.routerID["routerboard_address"])
	for _, k := range b.customKeys {
		out = append(out, b.customLabels[k])
	}
	return out
}

func (b *MetricBuilder) labelValsFromRecord(keys []string, record map[string]string) []string {
	extra := make([]string, len(keys))
	for i, k := range keys {
		extra[i] = record[k]
	}
	return b.labelVals(extra)
}

func desc(name, help string, labelNames []string) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "", name), help, labelNames, nil)
}

func (b *MetricBuilder) Gauge(ch chan<- prometheus.Metric, name, help, valueKey string, labelKeys []string, record map[string]string) {
	val := ParseFloat(record[valueKey])
	finalLabelVals := b.labelValsFromRecord(labelKeys, record)

	if b.isDuplicate(name, finalLabelVals) {
		return
	}
	metric, err := prometheus.NewConstMetric(
		desc(name, help, b.labelNames(labelKeys)),
		prometheus.GaugeValue,
		val,
		finalLabelVals...,
	)
	if err != nil {
		slog.Warn("Failed to build metric", "metric", name, "error", err)
		return
	}
	ch <- metric
}

func (b *MetricBuilder) GaugeVal(ch chan<- prometheus.Metric, name, help string, value float64, labelKeys []string, labelVals []string) {
	finalLabelVals := b.labelVals(labelVals)

	if b.isDuplicate(name, finalLabelVals) {
		return
	}
	metric, err := prometheus.NewConstMetric(
		desc(name, help, b.labelNames(labelKeys)),
		prometheus.GaugeValue,
		value,
		finalLabelVals...,
	)
	if err != nil {
		slog.Warn("Failed to build metric", "metric", name, "error", err)
		return
	}
	ch <- metric
}

func (b *MetricBuilder) Counter(ch chan<- prometheus.Metric, name, help, valueKey string, labelKeys []string, record map[string]string) {
	val := ParseFloat(record[valueKey])
	finalLabelVals := b.labelValsFromRecord(labelKeys, record)

	if b.isDuplicate(name, finalLabelVals) {
		return
	}
	metric, err := prometheus.NewConstMetric(
		desc(name, help, b.labelNames(labelKeys)),
		prometheus.CounterValue,
		val,
		finalLabelVals...,
	)
	if err != nil {
		slog.Warn("Failed to build metric", "metric", name, "error", err)
		return
	}
	ch <- metric
}

func (b *MetricBuilder) CounterVal(ch chan<- prometheus.Metric, name, help string, value float64, labelKeys []string, labelVals []string) {
	finalLabelVals := b.labelVals(labelVals)

	if b.isDuplicate(name, finalLabelVals) {
		return
	}
	metric, err := prometheus.NewConstMetric(
		desc(name, help, b.labelNames(labelKeys)),
		prometheus.CounterValue,
		value,
		finalLabelVals...,
	)
	if err != nil {
		slog.Warn("Failed to build metric", "metric", name, "error", err)
		return
	}
	ch <- metric
}

// Info emits a gauge=1 metric with all label values embedded in the label set.
// The metric name gets an "_info" suffix appended.
func (b *MetricBuilder) Info(ch chan<- prometheus.Metric, name, help string, labelKeys []string, record map[string]string) {
	fullMetricName := name + "_info"
	finalLabelVals := b.labelValsFromRecord(labelKeys, record)

	if b.isDuplicate(fullMetricName, finalLabelVals) {
		return
	}
	metric, err := prometheus.NewConstMetric(
		desc(name+"_info", help, b.labelNames(labelKeys)),
		prometheus.GaugeValue,
		1,
		finalLabelVals...,
	)
	if err != nil {
		slog.Warn("Failed to build metric", "metric", fullMetricName, "error", err)
		return
	}
	ch <- metric
}

func (b *MetricBuilder) isDuplicate(name string, labelVals []string) bool {
	digest := xxhash.New()
	for _, val := range labelVals {
		_, _ = digest.WriteString(val)
		_, _ = digest.WriteString("\x00")
	}
	hashKey := digest.Sum64()

	if _, ok := b.emitted[name]; !ok {
		b.emitted[name] = make(map[uint64]struct{})
	}

	if _, exists := b.emitted[name][hashKey]; exists {
		return true
	}

	b.emitted[name][hashKey] = struct{}{}
	return false
}
