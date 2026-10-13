package web

import (
	"crypto/subtle"
	"net/http"
	"strconv"
	"strings"

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
		httpRequestsTotal.WithLabelValues(pattern, methodLabel(r.Method), strconv.Itoa(rec.status)).Inc()
		httpRequestDuration.WithLabelValues(pattern).Observe(timer.ObserveDuration().Seconds())
	})
}

// methodLabel bounds the method label: the method is whatever the caller
// sent, and each new value would be a new time series kept forever.
func methodLabel(m string) string {
	switch m {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete, http.MethodOptions:
		return m
	default:
		return "OTHER"
	}
}

// MetricsEndpoint serves GET /metrics ahead of the API's own authentication,
// since a scrape token isn't a user token. With a token, a scrape must send
// "Authorization: Bearer <token>". Without one the endpoint is served openly
// when open is set (development) and not at all otherwise: route counts and
// latencies describe traffic, and aren't for anyone who asks.
func MetricsEndpoint(token string, open bool) Middleware {
	scrape := promhttp.Handler()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/metrics" || (r.Method != http.MethodGet && r.Method != http.MethodHead) ||
				(token == "" && !open) {
				next.ServeHTTP(w, r)
				return
			}
			if token != "" {
				got, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
				if !ok || subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
					w.Header().Set("WWW-Authenticate", `Bearer realm="metrics"`)
					RespondError(w, r, ErrUnauthorized(""))
					return
				}
			}
			scrape.ServeHTTP(w, r)
		})
	}
}
