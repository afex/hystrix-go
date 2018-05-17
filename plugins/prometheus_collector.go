package plugins

import (
	"github.com/afex/hystrix-go/hystrix/metric_collector"
	"github.com/prometheus/client_golang/prometheus"
)

// Constant namespace for metrics
const PROMETHEUS_NAMESPACE = "hystrix_go"

// This struct contains the metrics for prometheus. The handling of the values is completely done by the prometheus client library.
// The function `Collector` can be registered to the metricsCollector.Registry.
// If one want to use a custom registry it can be given via the reg parameter. If reg is nil, the prometheus default
// registry is used.
// The RunDuration is observed via a prometheus histogram ( https://prometheus.io/docs/concepts/metric_types/#histogram ).
// If the duration_buckets slice is nil, the "github.com/prometheus/client_golang/prometheus".DefBuckets  are used. As stated by the prometheus documentation, one should
// tailor the buckets to the response times of your application.
//
//
// Example use
//  package main
//
//  import (
//  	"github.com/afex/hystrix-go/plugins"
//  	"github.com/afex/hystrix-go/hystrix/metric_collector"
//  )
//
//  func main() {
//  	pc := plugins.NewPrometheusCollector(nil, nil)
//  	metricCollector.Registry.Register(pc.Collector)
//  }
type PrometheusCollector struct {
	attempts          *prometheus.CounterVec
	errors            *prometheus.CounterVec
	successes         *prometheus.CounterVec
	failures          *prometheus.CounterVec
	rejects           *prometheus.CounterVec
	shortCircuits     *prometheus.CounterVec
	timeouts          *prometheus.CounterVec
	fallbackSuccesses *prometheus.CounterVec
	fallbackFailures  *prometheus.CounterVec
	totalDuration     *prometheus.GaugeVec
	runDuration       *prometheus.HistogramVec
}

func NewPrometheusCollector(reg prometheus.Registerer, duration_buckets []float64) PrometheusCollector {
	if duration_buckets == nil {
		duration_buckets = prometheus.DefBuckets
	}
	hm := PrometheusCollector{
		attempts: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: PROMETHEUS_NAMESPACE,
			Name:      "attempts",
			Help:      "The number of updates.",
		}, []string{"command"}),
		errors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: PROMETHEUS_NAMESPACE,
			Name:      "errors",
			Help:      "The number of unsuccessful attempts. Attempts minus Errors will equal successes within a time range. Errors are any result from an attempt that is not a success.",
		}, []string{"command"}),
		successes: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: PROMETHEUS_NAMESPACE,
			Name:      "successes",
			Help:      "The number of requests that succeed.",
		}, []string{"command"}),
		failures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: PROMETHEUS_NAMESPACE,
			Name:      "failures",
			Help:      "The number of requests that fail.",
		}, []string{"command"}),
		rejects: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: PROMETHEUS_NAMESPACE,
			Name:      "rejects",
			Help:      "The number of requests that are rejected.",
		}, []string{"command"}),
		shortCircuits: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: PROMETHEUS_NAMESPACE,
			Name:      "short_circuits",
			Help:      "The number of requests that short circuited due to the circuit being open.",
		}, []string{"command"}),
		timeouts: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: PROMETHEUS_NAMESPACE,
			Name:      "timeouts",
			Help:      "The number of requests that are timeouted in the circuit breaker.",
		}, []string{"command"}),
		fallbackSuccesses: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: PROMETHEUS_NAMESPACE,
			Name:      "fallback_successes",
			Help:      "The number of successes that occurred during the execution of the fallback function.",
		}, []string{"command"}),
		fallbackFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: PROMETHEUS_NAMESPACE,
			Name:      "fallback_failures",
			Help:      "The number of failures that occurred during the execution of the fallback function.",
		}, []string{"command"}),
		totalDuration: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: PROMETHEUS_NAMESPACE,
			Name:      "total_duration_seconds",
			Help:      "The total runtime of this command in seconds.",
		}, []string{"command"}),
		runDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: PROMETHEUS_NAMESPACE,
			Name:      "run_duration_seconds",
			Help:      "Runtime of the Hystrix command.",
			Buckets:   duration_buckets,
		}, []string{"command"}),
	}
	if reg != nil {
		reg.MustRegister(
			hm.attempts,
			hm.errors,
			hm.failures,
			hm.rejects,
			hm.shortCircuits,
			hm.timeouts,
			hm.fallbackSuccesses,
			hm.fallbackFailures,
			hm.totalDuration,
			hm.runDuration,
		)
	} else {
		prometheus.MustRegister(
			hm.attempts,
			hm.errors,
			hm.failures,
			hm.rejects,
			hm.shortCircuits,
			hm.timeouts,
			hm.fallbackSuccesses,
			hm.fallbackFailures,
			hm.totalDuration,
			hm.runDuration,
		)
	}
	return hm
}

type cmdCollector struct {
	commandName string
	metrics     *PrometheusCollector
}

func (hc *cmdCollector) initCounters() {
	hc.metrics.attempts.WithLabelValues(hc.commandName).Add(0.0)
	hc.metrics.errors.WithLabelValues(hc.commandName).Add(0.0)
	hc.metrics.successes.WithLabelValues(hc.commandName).Add(0.0)
	hc.metrics.failures.WithLabelValues(hc.commandName).Add(0.0)
	hc.metrics.rejects.WithLabelValues(hc.commandName).Add(0.0)
	hc.metrics.shortCircuits.WithLabelValues(hc.commandName).Add(0.0)
	hc.metrics.timeouts.WithLabelValues(hc.commandName).Add(0.0)
	hc.metrics.fallbackSuccesses.WithLabelValues(hc.commandName).Add(0.0)
	hc.metrics.fallbackFailures.WithLabelValues(hc.commandName).Add(0.0)
	hc.metrics.totalDuration.WithLabelValues(hc.commandName).Set(0.0)
}

func (hm *PrometheusCollector) Collector(name string) metricCollector.MetricCollector {
	hc := &cmdCollector{
		commandName: name,
		metrics:     hm,
	}
	hc.initCounters()
	return hc
}

func (hc *cmdCollector) Update(result metricCollector.MetricResult) {
	hc.metrics.attempts.WithLabelValues(hc.commandName).Add(result.Attempts)
	hc.metrics.errors.WithLabelValues(hc.commandName).Add(result.Errors)
	hc.metrics.successes.WithLabelValues(hc.commandName).Add(result.Successes)
	hc.metrics.failures.WithLabelValues(hc.commandName).Add(result.Failures)
	hc.metrics.rejects.WithLabelValues(hc.commandName).Add(result.Rejects)
	hc.metrics.shortCircuits.WithLabelValues(hc.commandName).Add(result.ShortCircuits)
	hc.metrics.timeouts.WithLabelValues(hc.commandName).Add(result.Timeouts)
	hc.metrics.fallbackSuccesses.WithLabelValues(hc.commandName).Add(result.FallbackSuccesses)
	hc.metrics.fallbackFailures.WithLabelValues(hc.commandName).Add(result.FallbackFailures)
	hc.metrics.totalDuration.WithLabelValues(hc.commandName).Set(result.TotalDuration.Seconds())
	hc.metrics.runDuration.WithLabelValues(hc.commandName).Observe(result.TotalDuration.Seconds())
}

// Reset resets the internal counters and timers.
func (hc *cmdCollector) Reset() {
}
