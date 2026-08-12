package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// HTTP metrics (Prometheus-compatible exposition at GET /metrics).
var (
	reqTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests by method, route and status code.",
	}, []string{"method", "route", "status"})

	reqDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency by method and route.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route"})
)

// metricsResponseWriter captures the status code for metrics.
type metricsResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *metricsResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// observeMetrics wraps a handler with latency + counter instrumentation.
func (s *Server) observeMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		mw := &metricsResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(mw, r)
		reqTotal.WithLabelValues(r.Method, r.URL.Path, strconv.Itoa(mw.status)).Inc()
		reqDuration.WithLabelValues(r.Method, r.URL.Path).Observe(time.Since(start).Seconds())
	})
}

// handleMetrics exposes Prometheus metrics.
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	promhttp.Handler().ServeHTTP(w, r)
}
