package hystrix

import (
	"sync"
	"time"

	"github.com/afex/hystrix-go/hystrix/metric_collector"
	"github.com/afex/hystrix-go/hystrix/rolling"
)

type commandExecution struct {
	Type        string        `json:"type"`
	Start       time.Time     `json:"start_time"`
	RunDuration time.Duration `json:"run_duration"`
}

type metricExchange struct {
	Name    string
	Updates chan *commandExecution
	Mutex   *sync.RWMutex

	metricCollectors []metricCollector.MetricCollector
}

func newMetricExchange(name string) *metricExchange {
	m := &metricExchange{}
	m.Name = name

	m.Updates = make(chan *commandExecution)
	m.Mutex = &sync.RWMutex{}
	m.metricCollectors = metricCollector.Registry.InitializeMetricCollectors(name)
	m.Reset()

	go m.Monitor()

	return m
}

// The Default Collector function will panic if collectors are not setup to specification.
func (m *metricExchange) DefaultCollector() *metricCollector.DefaultMetricCollector {
	if len(m.metricCollectors) < 1 {
		panic("No Metric Collectors Registered.")
	}
	collection, ok := m.metricCollectors[0].(*metricCollector.DefaultMetricCollector)
	if !ok {
		panic("Default metric collector is not registered correctly. The default metric collector must be registered first.")
	}
	return collection
}

func (m *metricExchange) Monitor() {
	for update := range m.Updates {
		// we only grab a read lock to make sure Reset() isn't changing the numbers.
		m.Mutex.RLock()

		totalDuration := time.Now().Sub(update.Start)
		for _, collector := range m.metricCollectors {
			collector.IncrementAttempts()
			if update.Type != "success" {
				collector.IncrementErrors()
			}

			// granular metrics
			if update.Type == "success" {
				collector.IncrementSuccesses()
			}
			if update.Type == "failure" {
				collector.IncrementFailures()
			}
			if update.Type == "rejected" {
				collector.IncrementRejects()
			}
			if update.Type == "short-circuit" {
				collector.IncrementShortCircuits()
			}
			if update.Type == "timeout" {
				collector.IncrementTimeouts()
			}

			// fallback metrics
			if update.Type == "fallback-success" {
				collector.IncrementFallbackSuccesses()
			}
			if update.Type == "fallback-failure" {
				collector.IncrementFallbackFailures()
			}

			collector.UpdateTotalDuration(totalDuration)
			collector.UpdateRunDuration(update.RunDuration)
		}

		m.Mutex.RUnlock()
	}
}

func (m *metricExchange) Reset() {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	for _, collector := range m.metricCollectors {
		collector.Reset()
	}
}

func (m *metricExchange) Requests() *rolling.Number {
	m.Mutex.RLock()
	defer m.Mutex.RUnlock()

	return m.DefaultCollector().NumRequests
}

func (m *metricExchange) ErrorPercent(now time.Time) int {
	m.Mutex.RLock()
	defer m.Mutex.RUnlock()

	var errPct float64
	reqs := m.Requests().Sum(now)
	errs := m.DefaultCollector().Errors.Sum(now)

	if reqs > 0 {
		errPct = (float64(errs) / float64(reqs)) * 100
	}

	return int(errPct + 0.5)
}

func (m *metricExchange) IsHealthy(now time.Time) bool {
	return m.ErrorPercent(now) < getSettings(m.Name).ErrorPercentThreshold
}
