package hystrix

import (
	"sync"

	"github.com/afex/hystrix-go/hystrix/rolling"
)

type bufferedPoolMetrics struct {
	Mutex   *sync.RWMutex
	Updates chan bufferedPoolMetricsUpdate

	Name               string
	MaxActiveRequests  *rolling.Number
	MaxWaitingRequests *rolling.Number
	Executed           *rolling.Number
}

type bufferedPoolMetricsUpdate struct {
	activeCount  int
	waitingCount int
}

func newBufferedPoolMetrics(name string) *bufferedPoolMetrics {
	m := &bufferedPoolMetrics{}
	m.Name = name
	m.Updates = make(chan bufferedPoolMetricsUpdate)
	m.Mutex = &sync.RWMutex{}

	m.Reset()

	go m.Monitor()

	return m
}

func (m *bufferedPoolMetrics) Reset() {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	m.MaxActiveRequests = rolling.NewNumber()
	m.MaxWaitingRequests = rolling.NewNumber()
	m.Executed = rolling.NewNumber()
}

func (m *bufferedPoolMetrics) Monitor() {
	for u := range m.Updates {
		m.Mutex.RLock()

		m.Executed.Increment(1)
		m.MaxActiveRequests.UpdateMax(float64(u.activeCount))
		m.MaxWaitingRequests.UpdateMax(float64(u.waitingCount))

		m.Mutex.RUnlock()
	}
}
