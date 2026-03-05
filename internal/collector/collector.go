package collector

import (
	"context"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

// Collector is the interface that all metric collectors must implement.
// Describe may send nothing (unchecked collector pattern) to allow dynamic label sets.
type Collector interface {
	Name() string
	Describe(ch chan<- *prometheus.Desc)
	Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error
}

type Registry struct {
	collectors map[string]Collector
}

func NewRegistry() *Registry {
	return &Registry{
		collectors: make(map[string]Collector),
	}
}

func (r *Registry) Register(c Collector) {
	r.collectors[c.Name()] = c
}

func (r *Registry) Get(name string) Collector {
	return r.collectors[name]
}

func (r *Registry) All() []Collector {
	out := make([]Collector, 0, len(r.collectors))
	for _, c := range r.collectors {
		out = append(out, c)
	}
	return out
}
