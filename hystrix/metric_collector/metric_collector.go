package metric_collector

import (
	"sync"
	"time"
)

var Registry = metricCollectorRegistry{
	lock: &sync.RWMutex{},
	registry: []func(name string) MetricCollector{
		newDefaultMetricCollector,
	},
}

type metricCollectorRegistry struct {
	lock     *sync.RWMutex
	registry []func(name string) MetricCollector
}

func (m *metricCollectorRegistry) InitializeMetricCollectors(name string) []MetricCollector {
	m.lock.RLock()
	defer m.lock.RUnlock()

	metrics := make([]MetricCollector, len(m.registry))
	for i, metricCollectorInitializer := range m.registry {
		metrics[i] = metricCollectorInitializer(name)
	}
	return metrics
}

func (m *metricCollectorRegistry) Register(initMetricCollector func(string) MetricCollector) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.registry = append(m.registry, initMetricCollector)
}

type MetricCollector interface {
	// Always is incremented for each update.
	IncrementAttempts()
	// Always is incremented for each unsuccessful attempt.
	// Attempts minus Errors will equal successes within a time range.
	IncrementErrors()
	// Increments the number of requests that succeed.
	IncrementSuccesses()
	// Increments the number of requests that fail.
	IncrementFailures()
	// Increments the number of requests that are rejected.
	IncrementRejects()
	// Increments the number of requests that short circuited due to the circuit being open.
	IncrementShortCircuits()
	// Increments the number of timeouts that occurred in the circuit breaker.
	IncrementTimeouts()
	// Increments The number of successes that occurred during the execution of the fallback function.
	IncrementFallbackSuccesses()
	// Increments The number of failures that occurred during the execution of the fallback function.
	IncrementFallbackFailures()
	// Updates the internal counter of how long we've run for.
	UpdateTotalDuration(timeSinceStart time.Duration)
	// Updates the internal counter of how long the last run took.
	UpdateRunDuration(runDuration time.Duration)

	Reset()
}
