package hystrix

import (
	"sync"
	"time"

	"github.com/afex/hystrix-go/hystrix/metric_collector"
	"github.com/afex/hystrix-go/hystrix/rolling"
)

type commandExecution struct {
	Types       map[string]struct{} `json:"types"`
	Start       time.Time           `json:"start_time"`
	RunDuration time.Duration       `json:"run_duration"`
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

	m.Updates = make(chan *commandExecution, 2000)
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

		totalDuration := time.Since(update.Start)
		wg := &sync.WaitGroup{}
		for _, collector := range m.metricCollectors {
			wg.Add(1)
			go m.IncrementMetrics(wg, collector, update, totalDuration)
		}
		wg.Wait()

		m.Mutex.RUnlock()
	}
}

func isEventPresent(eventMap map[string]struct{}, key string) bool {
	_, present := eventMap[key]
	return present
}

func (m *metricExchange) IncrementMetrics(wg *sync.WaitGroup, collector metricCollector.MetricCollector, update *commandExecution, totalDuration time.Duration) {
	// granular metrics
	if isEventPresent(update.Types, "success") {
		collector.IncrementAttempts()
		collector.IncrementSuccesses()
	}
	if isEventPresent(update.Types, "failure") {
		collector.IncrementFailures()

		collector.IncrementAttempts()
		collector.IncrementErrors()
	}
	if isEventPresent(update.Types, "rejected") {
		collector.IncrementRejects()

		collector.IncrementAttempts()
		collector.IncrementErrors()
	}
	if isEventPresent(update.Types, "short-circuit") {
		collector.IncrementShortCircuits()

		collector.IncrementAttempts()
		collector.IncrementErrors()
	}
	if isEventPresent(update.Types, "timeout") {
		collector.IncrementTimeouts()

		collector.IncrementAttempts()
		collector.IncrementErrors()
	}
	if isEventPresent(update.Types, "queued") {
		collector.IncrementQueuedItem()
	}

	if len(update.Types) > 1 {
		// fallback metrics
		if isEventPresent(update.Types, "fallback-success") {
			collector.IncrementFallbackSuccesses()
		}
		if isEventPresent(update.Types, "fallback-failure") {
			collector.IncrementFallbackFailures()
		}
	}

	collector.UpdateTotalDuration(totalDuration)
	collector.UpdateRunDuration(update.RunDuration)

	wg.Done()
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
	return m.requestsLocked()
}

func (m *metricExchange) requestsLocked() *rolling.Number {
	return m.DefaultCollector().NumRequests()
}

func (m *metricExchange) ErrorPercent(now time.Time) int {
	m.Mutex.RLock()
	defer m.Mutex.RUnlock()

	var errPct float64
	reqs := m.requestsLocked().Sum(now)
	errs := m.DefaultCollector().Errors().Sum(now)

	if reqs > 0 {
		errPct = (errs / reqs) * 100.0
	}

	return int(errPct + 0.5)
}

func (m *metricExchange) IsHealthy(now time.Time) bool {
	return m.ErrorPercent(now) < getSettings(m.Name).ErrorPercentThreshold
}
