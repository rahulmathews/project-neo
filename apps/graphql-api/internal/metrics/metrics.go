package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// HTTP holds the HTTP-server-level Prometheus collectors.
type HTTP struct {
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	InFlight        prometheus.Gauge
}

// New registers a fresh HTTP collector set on the given registry.
func New(reg prometheus.Registerer) *HTTP {
	h := &HTTP{
		RequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "graphql_api",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total HTTP requests served, partitioned by route and status code.",
		}, []string{"method", "route", "status"}),
		RequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "graphql_api",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds, partitioned by route.",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		}, []string{"method", "route"}),
		InFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "graphql_api",
			Subsystem: "http",
			Name:      "in_flight_requests",
			Help:      "In-flight HTTP requests.",
		}),
	}
	reg.MustRegister(h.RequestsTotal, h.RequestDuration, h.InFlight)
	return h
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
