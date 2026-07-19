package web

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "adera_http_requests_total",
		Help: "HTTP requests by route pattern, method, and status.",
	}, []string{"pattern", "method", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "adera_http_request_duration_seconds",
		Help:    "HTTP request latency by route pattern.",
		Buckets: prometheus.DefBuckets,
	}, []string{"pattern"})

	httpInFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "adera_http_requests_in_flight",
		Help: "HTTP requests currently being served.",
	})
)

// Metrics records per-request metrics keyed by the matched route pattern
// (never the raw path, which would explode cardinality).
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpInFlight.Inc()
		defer httpInFlight.Dec()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		timer := prometheus.NewTimer(nil)
		next.ServeHTTP(rec, r)
		pattern := r.Pattern
		if pattern == "" {
			pattern = "unmatched"
		}
		httpRequestsTotal.WithLabelValues(pattern, r.Method, strconv.Itoa(rec.status)).Inc()
		httpRequestDuration.WithLabelValues(pattern).Observe(timer.ObserveDuration().Seconds())
	})
}

// MetricsHandler serves the Prometheus scrape endpoint.
func MetricsHandler() http.Handler { return promhttp.Handler() }
