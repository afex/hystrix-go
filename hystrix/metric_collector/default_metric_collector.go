package metric_collector

import (
	"time"

	"github.com/afex/hystrix-go/hystrix/rolling"
)

type DefaultMetricCollector struct {
	NumRequests *rolling.RollingNumber
	Errors      *rolling.RollingNumber

	Successes     *rolling.RollingNumber
	Failures      *rolling.RollingNumber
	Rejects       *rolling.RollingNumber
	ShortCircuits *rolling.RollingNumber
	Timeouts      *rolling.RollingNumber

	FallbackSuccesses *rolling.RollingNumber
	FallbackFailures  *rolling.RollingNumber
	TotalDuration     *rolling.RollingTiming
	RunDuration       *rolling.RollingTiming
}

func newDefaultMetricCollector(name string) MetricCollector {
	m := &DefaultMetricCollector{}
	m.Reset()
	return m
}

func (d *DefaultMetricCollector) IncrementAttempts() {
	d.NumRequests.Increment()
}

func (d *DefaultMetricCollector) IncrementErrors() {
	d.Errors.Increment()
}

func (d *DefaultMetricCollector) IncrementSuccesses() {
	d.Successes.Increment()
}

func (d *DefaultMetricCollector) IncrementFailures() {
	d.Failures.Increment()
}

func (d *DefaultMetricCollector) IncrementRejects() {
	d.Rejects.Increment()
}

func (d *DefaultMetricCollector) IncrementShortCircuits() {
	d.ShortCircuits.Increment()
}

func (d *DefaultMetricCollector) IncrementTimeouts() {
	d.Timeouts.Increment()
}

func (d *DefaultMetricCollector) IncrementFallbackSuccesses() {
	d.FallbackSuccesses.Increment()
}

func (d *DefaultMetricCollector) IncrementFallbackFailures() {
	d.FallbackFailures.Increment()
}

func (d *DefaultMetricCollector) UpdateTotalDuration(timeSinceStart time.Duration) {
	d.TotalDuration.Add(timeSinceStart)
}

func (d *DefaultMetricCollector) UpdateRunDuration(runDuration time.Duration) {
	d.RunDuration.Add(runDuration)
}

func (d *DefaultMetricCollector) Reset() {
	d.NumRequests = rolling.NewRollingNumber()
	d.Errors = rolling.NewRollingNumber()
	d.Successes = rolling.NewRollingNumber()
	d.Rejects = rolling.NewRollingNumber()
	d.ShortCircuits = rolling.NewRollingNumber()
	d.Failures = rolling.NewRollingNumber()
	d.Timeouts = rolling.NewRollingNumber()
	d.FallbackSuccesses = rolling.NewRollingNumber()
	d.FallbackFailures = rolling.NewRollingNumber()
	d.TotalDuration = rolling.NewRollingTiming()
	d.RunDuration = rolling.NewRollingTiming()
}
