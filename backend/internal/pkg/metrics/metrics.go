// Package metrics centralizes Prometheus instrumentation: an HTTP middleware,
// runtime collectors, breaker state gauge and business counters.
package metrics

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	reg = prometheus.NewRegistry()

	httpRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "laundrypro_http_requests_total",
		Help: "Total HTTP requests by method, path and status.",
	}, []string{"method", "path", "status"})

	httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "laundrypro_http_request_duration_seconds",
		Help:    "HTTP request latency by method and path.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	breakerState = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "laundrypro_breaker_state",
		Help: "Circuit breaker state: closed=0 half-open=1 open=2.",
	}, []string{"breaker"})

	breakerCalls = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "laundrypro_breaker_calls_total",
		Help: "Circuit breaker call outcomes.",
	}, []string{"breaker", "result"})

	breakerLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "laundrypro_breaker_call_duration_seconds",
		Help:    "Circuit breaker call latency.",
		Buckets: prometheus.DefBuckets,
	}, []string{"breaker"})

	outboxPending = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "laundrypro_outbox_pending_events",
		Help: "Number of outbox events awaiting processing.",
	})

	outboxProcessed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "laundrypro_outbox_processed_total",
		Help: "Outbox events processed by outcome.",
	}, []string{"outcome"})

	cacheHits = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "laundrypro_cache_operations_total",
		Help: "Cache operations by kind and result.",
	}, []string{"kind", "result"})
)

func init() {
	reg.MustRegister(httpRequests, httpDuration, breakerState, breakerCalls, breakerLatency, outboxPending, outboxProcessed, cacheHits)
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
}

// Handler returns the Prometheus scrape handler (GET /metrics).
func Handler() http.Handler {
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
}

// Middleware instruments HTTP requests.
func Middleware() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		start := time.Now()
		c.Next(ctx)
		path := string(c.Path())
		if len(path) > 80 {
			path = path[:80]
		}
		status := strconv.Itoa(c.Response.StatusCode())
		httpRequests.WithLabelValues(string(c.Method()), path, status).Inc()
		httpDuration.WithLabelValues(string(c.Method()), path).Observe(time.Since(start).Seconds())
	}
}

// SetBreakerState records a breaker state transition.
func SetBreakerState(name, from, to string) {
	breakerState.WithLabelValues(name).Set(stateValue(to))
}

func stateValue(state string) float64 {
	switch state {
	case "half-open":
		return 1
	case "open":
		return 2
	default:
		return 0
	}
}

// ObserveBreakerCall records breaker call outcome and latency.
func ObserveBreakerCall(name string, d time.Duration, ok bool) {
	result := "success"
	if !ok {
		result = "failure"
	}
	breakerCalls.WithLabelValues(name, result).Inc()
	breakerLatency.WithLabelValues(name).Observe(d.Seconds())
}

// SetOutboxPending updates the pending outbox gauge.
func SetOutboxPending(n int) { outboxPending.Set(float64(n)) }

// ObserveOutbox records outbox processing outcome.
func ObserveOutbox(outcome string) { outboxProcessed.WithLabelValues(outcome).Inc() }

// ObserveCache records a cache operation result.
func ObserveCache(kind string, hit bool) {
	result := "miss"
	if hit {
		result = "hit"
	}
	cacheHits.WithLabelValues(kind, result).Inc()
}
