package hystrix_go

import (
	"sync"
	"time"

	metricCollector "github.com/afex/hystrix-go/hystrix/metric_collector"
	"github.com/prometheus/client_golang/prometheus"
)

type PrometheusCollector struct {
	sync.RWMutex
	circuitOpen             *prometheus.GaugeVec
	attempts                *prometheus.CounterVec
	errors                  *prometheus.CounterVec
	successes               *prometheus.CounterVec
	failures                *prometheus.CounterVec
	rejects                 *prometheus.CounterVec
	shortCircuits           *prometheus.CounterVec
	timeouts                *prometheus.CounterVec
	fallbackSuccesses       *prometheus.CounterVec
	fallbackFailures        *prometheus.CounterVec
	contextCanceled         *prometheus.CounterVec
	contextDeadlineExceeded *prometheus.CounterVec
	totalDuration           *prometheus.HistogramVec
	runDuration             *prometheus.HistogramVec
	concurrencyInUse        *prometheus.GaugeVec
}

func NewPrometheusCollector(namespace string, reg prometheus.Registerer, durationBuckets []float64) *PrometheusCollector {
	if namespace == "" {
		namespace = "hystrix"
	}
	if durationBuckets == nil {
		durationBuckets = prometheus.DefBuckets
	}
	pc := &PrometheusCollector{
		circuitOpen:             CircuitOpenCommand(namespace),
		attempts:                AttemptsTotalCommand(namespace),
		errors:                  ErrorsTotalCommand(namespace),
		successes:               SuccessTotalCommand(namespace),
		failures:                FailureTotalCommand(namespace),
		rejects:                 RejectTotalCommand(namespace),
		shortCircuits:           ShortCircuitTotalCommand(namespace),
		timeouts:                TimeoutsTotalCommand(namespace),
		fallbackSuccesses:       FallbackSuccessesTotalCommand(namespace),
		fallbackFailures:        FallbackFailuresTotalCommand(namespace),
		contextCanceled:         ContextCanceledTotalCommand(namespace),
		contextDeadlineExceeded: ContextDeadlineExceededTotalCommand(namespace),
		totalDuration:           TotalDurationSecondsCommand(namespace, durationBuckets),
		runDuration:             RunDurationSecondsCommand(namespace, durationBuckets),
		concurrencyInUse:        ConcurrencyInUseCommand(namespace),
	}
	if reg != nil {
		reg.MustRegister(
			pc.circuitOpen,
			pc.attempts,
			pc.errors,
			pc.successes,
			pc.failures,
			pc.rejects,
			pc.shortCircuits,
			pc.timeouts,
			pc.fallbackSuccesses,
			pc.fallbackFailures,
			pc.contextCanceled,
			pc.contextDeadlineExceeded,
			pc.totalDuration,
			pc.runDuration,
			pc.concurrencyInUse,
		)
	} else {
		prometheus.MustRegister(
			pc.circuitOpen,
			pc.attempts,
			pc.errors,
			pc.successes,
			pc.failures,
			pc.rejects,
			pc.shortCircuits,
			pc.timeouts,
			pc.fallbackSuccesses,
			pc.fallbackFailures,
			pc.contextCanceled,
			pc.contextDeadlineExceeded,
			pc.totalDuration,
			pc.runDuration,
			pc.concurrencyInUse,
		)
	}
	return pc
}

func RunDurationSecondsCommand(namespace string, durationBuckets []float64) *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "run_duration_seconds",
		Help:      "The duration of Hystrix command execution. This only measure the command only, without Hystrix overhead.",
		Buckets:   durationBuckets,
	}, []string{"command"})
}

func TotalDurationSecondsCommand(namespace string, durationBuckets []float64) *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "total_duration_seconds",
		Help: "The total duration, includes thread queuing/scheduling/execution, semaphores, " +
			"circuit breaker logic, and other aspects of overhead, of a Hystrix command.",
		Buckets: durationBuckets,
	}, []string{"command"})
}

func ContextDeadlineExceededTotalCommand(namespace string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "context_deadline_exceeded_total",
		Help:      "The number of context deadline exceeded.",
	}, []string{"command"})
}

func ContextCanceledTotalCommand(namespace string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "context_canceled_total",
		Help:      "The number of context canceled.",
	}, []string{"command"})
}

func FallbackFailuresTotalCommand(namespace string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "fallback_failures_total",
		Help:      "The number of failures that occurred during the execution of the fallback function.",
	}, []string{"command"})
}

func FallbackSuccessesTotalCommand(namespace string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "fallback_successes_total",
		Help:      "The number of successes that occurred during the execution of the fallback function.",
	}, []string{"command"})
}

func TimeoutsTotalCommand(namespace string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "timeouts_total",
		Help:      "The number of requests that are timeouted in the circuit breaker.",
	}, []string{"command"})
}

func ShortCircuitTotalCommand(namespace string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "short_circuits_total",
		Help:      "The number of requests that short circuited due to the circuit being open.",
	}, []string{"command"})
}

func RejectTotalCommand(namespace string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "rejects_total",
		Help:      "The number of requests that are rejected.",
	}, []string{"command"})
}

func FailureTotalCommand(namespace string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "failures_total",
		Help:      "The number of requests that fail.",
	}, []string{"command"})
}

func SuccessTotalCommand(namespace string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "successes_total",
		Help:      "The number of requests that succeed.",
	}, []string{"command"})
}

func ErrorsTotalCommand(namespace string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "errors_total",
		Help: "The number of unsuccessful attempts. Attempts minus Errors will equal successes within a time range. " +
			"Errors are any result from an attempt that is not a success.",
	}, []string{"command"})
}

func AttemptsTotalCommand(namespace string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "attempts_total",
		Help:      "The number of requests.",
	}, []string{"command"})
}

func CircuitOpenCommand(namespace string) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "circuit_open",
		Help:      "Status of the circuit. Zero value means a closed circuit.",
	}, []string{"command"})
}

func ConcurrencyInUseCommand(namespace string) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "concurrency_in_use",
		Help:      "Current concurrency of Hystrix command in use",
	}, []string{"command"})
}

type cmdCollector struct {
	commandName string
	metrics     *PrometheusCollector
}

func (pc *PrometheusCollector) Collector(name string) metricCollector.MetricCollector {
	c := &cmdCollector{
		commandName: name,
		metrics:     pc,
	}
	c.initCounters()
	return c
}

func (c *cmdCollector) initCounters() {
	c.metrics.circuitOpen.WithLabelValues(c.commandName).Set(float64(Zero))
	c.metrics.attempts.WithLabelValues(c.commandName).Add(float64(Zero))
	c.metrics.errors.WithLabelValues(c.commandName).Add(float64(Zero))
	c.metrics.successes.WithLabelValues(c.commandName).Add(float64(Zero))
	c.metrics.failures.WithLabelValues(c.commandName).Add(float64(Zero))
	c.metrics.rejects.WithLabelValues(c.commandName).Add(float64(Zero))
	c.metrics.shortCircuits.WithLabelValues(c.commandName).Add(float64(Zero))
	c.metrics.timeouts.WithLabelValues(c.commandName).Add(Zero)
	c.metrics.fallbackSuccesses.WithLabelValues(c.commandName).Add(float64(Zero))
	c.metrics.fallbackFailures.WithLabelValues(c.commandName).Add(Zero)
	c.metrics.contextCanceled.WithLabelValues(c.commandName).Add(Zero)
	c.metrics.contextDeadlineExceeded.WithLabelValues(c.commandName).Add(Zero)
	c.metrics.concurrencyInUse.WithLabelValues(c.commandName).Set(float64(Zero))
}

func (c *cmdCollector) setGaugeMetric(pg *prometheus.GaugeVec, i float64) {
	pg.WithLabelValues(c.commandName).Set(i)
}

func (c *cmdCollector) incrementCounterMetric(pc *prometheus.CounterVec, i float64) {
	if i == Zero {
		return
	}
	pc.WithLabelValues(c.commandName).Add(i)
}

func (c *cmdCollector) updateTimerMetric(ph *prometheus.HistogramVec, dur time.Duration) {
	ph.WithLabelValues(c.commandName).Observe(dur.Seconds())
}

func (c *cmdCollector) Update(r metricCollector.MetricResult) {
	c.metrics.RWMutex.Lock()
	defer c.metrics.RWMutex.Unlock()

	if r.Successes > Zero {
		c.setGaugeMetric(c.metrics.circuitOpen, Zero)
		c.setGaugeMetric(c.metrics.concurrencyInUse, Zero)
	} else if r.ShortCircuits > Zero {
		c.setGaugeMetric(c.metrics.circuitOpen, One)
		c.setGaugeMetric(c.metrics.concurrencyInUse, One)
	}

	c.incrementCounterMetric(c.metrics.attempts, r.Attempts)
	c.incrementCounterMetric(c.metrics.errors, r.Errors)
	c.incrementCounterMetric(c.metrics.successes, r.Successes)
	c.incrementCounterMetric(c.metrics.failures, r.Failures)
	c.incrementCounterMetric(c.metrics.rejects, r.Rejects)
	c.incrementCounterMetric(c.metrics.shortCircuits, r.ShortCircuits)
	c.incrementCounterMetric(c.metrics.timeouts, r.Timeouts)
	c.incrementCounterMetric(c.metrics.fallbackSuccesses, r.FallbackSuccesses)
	c.incrementCounterMetric(c.metrics.fallbackFailures, r.FallbackFailures)
	c.incrementCounterMetric(c.metrics.contextCanceled, r.ContextCanceled)
	c.incrementCounterMetric(c.metrics.contextDeadlineExceeded, r.ContextDeadlineExceeded)

	c.updateTimerMetric(c.metrics.totalDuration, r.TotalDuration)
	c.updateTimerMetric(c.metrics.runDuration, r.RunDuration)
}

// Reset is a noop operation in this collector.
func (c *cmdCollector) Reset() {}
