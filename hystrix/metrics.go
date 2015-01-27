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
	Name    string
	Updates chan *ExecutionMetric
	Mutex   *sync.RWMutex

	numRequests *rollingNumber
	Errors      *rollingNumber

	Successes     *rollingNumber
	Failures      *rollingNumber
	Rejected      *rollingNumber
	ShortCircuits *rollingNumber
	Timeouts      *rollingNumber

	FallbackSuccesses *rollingNumber
	FallbackFailures  *rollingNumber

	TotalDuration *rollingTiming
	RunDuration   *rollingTiming
}

func NewMetrics(name string) *Metrics {
	m := &Metrics{}
	m.Name = name

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
		m.numRequests.Increment()
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

	m.numRequests = newRollingNumber()
	m.Errors = newRollingNumber()

	m.Successes = newRollingNumber()
	m.Rejected = newRollingNumber()
	m.ShortCircuits = newRollingNumber()
	m.Failures = newRollingNumber()
	m.Timeouts = newRollingNumber()

	m.FallbackSuccesses = newRollingNumber()
	m.FallbackFailures = newRollingNumber()

	m.TotalDuration = newRollingTiming()
	m.RunDuration = newRollingTiming()
}

func (m *Metrics) Requests() *rollingNumber {
	m.Mutex.RLock()
	defer m.Mutex.RUnlock()

	return m.numRequests
}

func (m *Metrics) ErrorPercent(now time.Time) int {
	m.Mutex.RLock()
	defer m.Mutex.RUnlock()

	var errPct float64
	reqs := m.Requests().Sum(now)
	errs := m.Errors.Sum(now)

	if reqs > 0 {
		errPct = (float64(errs) / float64(reqs)) * 100
	}

	return int(errPct + 0.5)
}

func (m *Metrics) IsHealthy(now time.Time) bool {
	return m.ErrorPercent(now) < GetErrorPercentThreshold(m.Name)
}
