package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// HTTP holds workers' HTTP-server-level Prometheus collectors.
type HTTP struct {
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
}

// Parser holds parser-pipeline collectors.
type Parser struct {
	// Messages: outcome label = success|failed|skipped
	Messages *prometheus.CounterVec
	// Extractor: provider = regex|llm; outcome = matched|miss|success|not_a_ride|error
	Extractor *prometheus.CounterVec
	// ExtractDuration: provider = regex|llm
	ExtractDuration *prometheus.HistogramVec
	Retries         prometheus.Counter
}

// New registers HTTP + Parser collectors on the given registry.
func New(reg prometheus.Registerer) (*HTTP, *Parser) {
	h := &HTTP{
		RequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "workers",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total HTTP requests served, partitioned by route and status.",
		}, []string{"method", "route", "status"}),
		RequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "workers",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds, partitioned by route.",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		}, []string{"method", "route"}),
	}
	p := &Parser{
		Messages: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "workers",
			Subsystem: "parser",
			Name:      "messages_total",
			Help:      "Total messages processed by the parser, by terminal outcome.",
		}, []string{"outcome"}),
		Extractor: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "workers",
			Subsystem: "parser",
			Name:      "extractor_total",
			Help:      "Extraction attempts by provider and outcome.",
		}, []string{"provider", "outcome"}),
		ExtractDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "workers",
			Subsystem: "parser",
			Name:      "extract_duration_seconds",
			Help:      "Extraction latency by provider.",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
		}, []string{"provider"}),
		Retries: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "workers",
			Subsystem: "parser",
			Name:      "retries_total",
			Help:      "Total retry attempts triggered by transient failures.",
		}),
	}
	reg.MustRegister(
		h.RequestsTotal, h.RequestDuration,
		p.Messages, p.Extractor, p.ExtractDuration, p.Retries,
	)
	return h, p
}

// NewRegistry returns a registry seeded with Go runtime + process collectors.
func NewRegistry() *prometheus.Registry {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return reg
}

// Handler returns the /metrics HTTP handler bound to the given registry.
func Handler(reg *prometheus.Registry) http.Handler {
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg})
}
