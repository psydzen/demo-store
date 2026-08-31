// Package obs holds the request metrics every RPC and every HTTP handler is
// expected to report.
package obs

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Requests counts finished calls by method and outcome.
	Requests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "quiz_requests_total",
		Help: "Finished calls, by method and outcome.",
	}, []string{"method", "outcome"})

	// Latency records how long a call took, by method.
	Latency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "quiz_request_duration_seconds",
		Help:    "Call duration, by method.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method"})
)

// Observe starts the metric for a call. The returned function must be called
// when the call finishes; it records both the counter and the duration.
//
// A handler that never reaches Observe reports no metrics at all, which is what
// the "every request reports metrics" rule looks for.
func Observe(ctx context.Context, method string) func(err error) {
	started := time.Now()
	return func(err error) {
		outcome := "ok"
		if err != nil {
			outcome = "error"
		}
		Requests.WithLabelValues(method, outcome).Inc()
		Latency.WithLabelValues(method).Observe(time.Since(started).Seconds())
	}
}
