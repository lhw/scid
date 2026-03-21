package api

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	metricRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scid_http_requests_total",
			Help: "Total HTTP requests partitioned by method, route pattern, and status code.",
		},
		[]string{"method", "route", "status"},
	)
	metricRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "scid_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)
	// metricVerifications counts RSI handle verification outcomes.
	// Labels: "success" | "token_mismatch" | "fetch_error" | "error"
	metricVerifications = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scid_verifications_total",
			Help: "RSI handle verification outcomes.",
		},
		[]string{"result"},
	)
	// metricOrgSyncs counts background org sync runs per user.
	metricOrgSyncs = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scid_org_syncs_total",
			Help: "Background org sync runs per user.",
		},
		[]string{"result"}, // "synced"
	)
	// metricSignupTokens counts one-use Pocket ID signup tokens issued.
	metricSignupTokens = promauto.NewCounter(prometheus.CounterOpts{
		Name: "scid_signup_tokens_total",
		Help: "Pocket ID one-use signup tokens issued.",
	})
)

// auditLog emits a structured slog event tagged with the action name so that
// log-aggregation tools (Loki, Datadog, etc.) can filter audit-trail events
// with a simple query like `{action=~"rsi.*"}`.
func auditLog(ctx context.Context, action string, pairs ...any) {
	slog.InfoContext(ctx, "audit", append([]any{"action", action}, pairs...)...)
}

// prometheusMiddleware records per-request HTTP metrics. It uses chi's
// route pattern (e.g. "/api/apps/{id}") rather than the raw URL path to
// avoid high-cardinality Prometheus label explosions from dynamic IDs.
//
// chi mutates the *chi.Context in place during handler dispatch, so by the
// time next.ServeHTTP returns the route pattern is already set in the
// context that this middleware holds.
func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)

		route := r.URL.Path
		if rctx := chi.RouteContext(r.Context()); rctx != nil {
			if p := rctx.RoutePattern(); p != "" {
				route = p
			}
		}

		metricRequestsTotal.With(prometheus.Labels{
			"method": r.Method,
			"route":  route,
			"status": strconv.Itoa(rw.status),
		}).Inc()
		metricRequestDuration.With(prometheus.Labels{
			"method": r.Method,
			"route":  route,
		}).Observe(time.Since(start).Seconds())
	})
}

// statusRecorder wraps http.ResponseWriter to capture the written status code.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// prometheusHandler returns the default Prometheus text exposition handler.
// Served at /metrics (internal port only, not proxied through Caddy).
func prometheusHandler() http.Handler {
	return promhttp.Handler()
}
