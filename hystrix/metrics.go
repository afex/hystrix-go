package hystrix

import (
	"sync"
	"time"
)

type ExecutionMetric struct {
	Type          string
	Time          time.Time
	TotalDuration time.Duration
	RunDuration   time.Duration
}

type Metrics struct {
	Updates chan *ExecutionMetric
	Mutex   *sync.RWMutex

	Requests *RollingNumber
	Errors   *RollingNumber

	Successes     *RollingNumber
	Failures      *RollingNumber
	Rejected      *RollingNumber
	ShortCircuits *RollingNumber
	Timeouts      *RollingNumber

	FallbackSuccesses *RollingNumber
	FallbackFailures  *RollingNumber

	TotalDuration *RollingTiming
	RunDuration   *RollingTiming
}

func NewMetrics() *Metrics {
	m := &Metrics{}

	m.Updates = make(chan *ExecutionMetric)
	m.Mutex = &sync.RWMutex{}

	m.Reset()

	go m.Monitor()

	return m
}

func (m *Metrics) Monitor() {
	for update := range m.Updates {
		// we only grab a read lock to make sure Reset() isn't changing the numbers
		// Increment() and Add() implement their own internal locking
		m.Mutex.RLock()

		// combined metrics
		m.Requests.Increment()
		if update.Type != "success" {
			m.Errors.Increment()
		}

		// granular metrics
		if update.Type == "success" {
			m.Successes.Increment()
		}
		if update.Type == "failure" {
			m.Failures.Increment()
		}
		if update.Type == "rejected" {
			m.Rejected.Increment()
		}
		if update.Type == "short-circuit" {
			m.ShortCircuits.Increment()
		}
		if update.Type == "timeout" {
			m.Timeouts.Increment()
		}

		// fallback metrics
		if update.Type == "fallback-success" {
			m.FallbackSuccesses.Increment()
		}
		if update.Type == "fallback-failure" {
			m.FallbackFailures.Increment()
		}

		m.TotalDuration.Add(update.TotalDuration)
		m.RunDuration.Add(update.RunDuration)

		m.Mutex.RUnlock()
	}
}

func (m *Metrics) Reset() {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	m.Requests = NewRollingNumber()
	m.Errors = NewRollingNumber()

	m.Successes = NewRollingNumber()
	m.Rejected = NewRollingNumber()
	m.ShortCircuits = NewRollingNumber()
	m.Failures = NewRollingNumber()
	m.Timeouts = NewRollingNumber()

	m.FallbackSuccesses = NewRollingNumber()
	m.FallbackFailures = NewRollingNumber()

	m.TotalDuration = NewRollingTiming()
	m.RunDuration = NewRollingTiming()
}

func (m *Metrics) ErrorPercent(now time.Time) float64 {
	m.Mutex.RLock()
	defer m.Mutex.RUnlock()

	var errPct float64
	reqs := m.Requests.Sum(now)
	errs := m.Errors.Sum(now)

	if reqs > 0 {
		errPct = float64(errs) / float64(reqs)
	}

	return errPct
}

func (m *Metrics) IsHealthy(now time.Time) bool {
	// TODO: configurable error percent threshold
	return m.ErrorPercent(now) < 0.50
}
