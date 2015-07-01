package metricCollector

import (
	"time"

	"github.com/afex/hystrix-go/hystrix/rolling"
)

// DefaultMetricCollector holds information about the circuit state.
// This implementation of MetricCollector is the canonical source of information about the circuit.
// It is used for for all internal hystrix operations
// including circuit health checks and metrics sent to the hystrix dashboard.
//
// Metric Collectors do not need Mutexes as they are updated by circuits within a locked context.
type DefaultMetricCollector struct {
	NumRequests *rolling.Number
	Errors      *rolling.Number

	Successes     *rolling.Number
	Failures      *rolling.Number
	Rejects       *rolling.Number
	ShortCircuits *rolling.Number
	Timeouts      *rolling.Number

	FallbackSuccesses *rolling.Number
	FallbackFailures  *rolling.Number
	TotalDuration     *rolling.Timing
	RunDuration       *rolling.Timing
}

func newDefaultMetricCollector(name string) MetricCollector {
	m := &DefaultMetricCollector{}
	m.Reset()
	return m
}

// IncrementAttempts increments the number of requests seen in the latest time bucket.
func (d *DefaultMetricCollector) IncrementAttempts() {
	d.NumRequests.Increment(1)
}

// IncrementErrors increments the number of errors seen in the latest time bucket.
// Errors are any result from an attempt that is not a success.
func (d *DefaultMetricCollector) IncrementErrors() {
	d.Errors.Increment(1)
}

// IncrementSuccesses increments the number of successes seen in the latest time bucket.
func (d *DefaultMetricCollector) IncrementSuccesses() {
	d.Successes.Increment(1)
}

// IncrementFailures increments the number of failures seen in the latest time bucket.
func (d *DefaultMetricCollector) IncrementFailures() {
	d.Failures.Increment(1)
}

// IncrementRejects increments the number of rejected requests seen in the latest time bucket.
func (d *DefaultMetricCollector) IncrementRejects() {
	d.Rejects.Increment(1)
}

// IncrementShortCircuits increments the number of rejected requests seen in the latest time bucket.
func (d *DefaultMetricCollector) IncrementShortCircuits() {
	d.ShortCircuits.Increment(1)
}

// IncrementTimeouts increments the number of requests that timed out in the latest time bucket.
func (d *DefaultMetricCollector) IncrementTimeouts() {
	d.Timeouts.Increment(1)
}

// IncrementFallbackSuccesses increments the number of successful calls to the fallback function in the latest time bucket.
func (d *DefaultMetricCollector) IncrementFallbackSuccesses() {
	d.FallbackSuccesses.Increment(1)
}

// IncrementFallbackFailures increments the number of failed calls to the fallback function in the latest time bucket.
func (d *DefaultMetricCollector) IncrementFallbackFailures() {
	d.FallbackFailures.Increment(1)
}

// UpdateTotalDuration updates the total amount of time this circuit has been running.
func (d *DefaultMetricCollector) UpdateTotalDuration(timeSinceStart time.Duration) {
	d.TotalDuration.Add(timeSinceStart)
}

// UpdateRunDuration updates the amount of time the latest request took to complete.
func (d *DefaultMetricCollector) UpdateRunDuration(runDuration time.Duration) {
	d.RunDuration.Add(runDuration)
}

// Reset resets all metrics in this collector to 0.
func (d *DefaultMetricCollector) Reset() {
	d.NumRequests = rolling.NewNumber()
	d.Errors = rolling.NewNumber()
	d.Successes = rolling.NewNumber()
	d.Rejects = rolling.NewNumber()
	d.ShortCircuits = rolling.NewNumber()
	d.Failures = rolling.NewNumber()
	d.Timeouts = rolling.NewNumber()
	d.FallbackSuccesses = rolling.NewNumber()
	d.FallbackFailures = rolling.NewNumber()
	d.TotalDuration = rolling.NewTiming()
	d.RunDuration = rolling.NewTiming()
}
