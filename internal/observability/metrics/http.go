package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type HTTPMetrics struct {
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	inFlightRequests prometheus.Gauge
}

func NewHTTPMetrics(reg prometheus.Registerer) *HTTPMetrics {
	m := &HTTPMetrics{
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "pokemon_ai",
				Subsystem: "http",
				Name:      "requests_total",
				Help:      "Total HTTP requests.",
			},
			[]string{"route", "status"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "pokemon_ai",
				Subsystem: "http",
				Name:      "request_duration_seconds",
				Help:      "HTTP request duration in seconds.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"route", "status"},
		),
		inFlightRequests: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "pokemon_ai",
				Subsystem: "http",
				Name:      "in_flight_requests",
				Help:      "Current number of in-flight HTTP requests.",
			},
		),
	}
	reg.MustRegister(m.requestsTotal, m.requestDuration, m.inFlightRequests)
	return m
}

func (m *HTTPMetrics) ObserveRequest(route string, statusCode int, duration time.Duration) {
	status := strconv.Itoa(statusCode)
	m.requestsTotal.WithLabelValues(route, status).Inc()
	m.requestDuration.WithLabelValues(route, status).Observe(duration.Seconds())
}

func (m *HTTPMetrics) IncInFlight() {
	m.inFlightRequests.Inc()
}

func (m *HTTPMetrics) DecInFlight() {
	m.inFlightRequests.Dec()
}

type statusWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewStatusWriter(w http.ResponseWriter) *statusWriter {
	return &statusWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

func (w *statusWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *statusWriter) StatusCode() int {
	return w.statusCode
}
