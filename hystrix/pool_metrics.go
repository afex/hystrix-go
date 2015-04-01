package hystrix

import (
	"sync"

	"github.com/afex/hystrix-go/hystrix/rolling"
)

type poolMetrics struct {
	Mutex   *sync.RWMutex
	Updates chan poolMetricsUpdate

	Name              string
	MaxActiveRequests *rolling.RollingNumber
	Executed          *rolling.RollingNumber
}

type poolMetricsUpdate struct {
	activeCount int
}

func newPoolMetrics(name string) *poolMetrics {
	m := &poolMetrics{}
	m.Name = name
	m.Updates = make(chan poolMetricsUpdate)
	m.Mutex = &sync.RWMutex{}

	m.Reset()

	go m.Monitor()

	return m
}

func (m *poolMetrics) Reset() {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	m.MaxActiveRequests = rolling.NewRollingNumber()
	m.Executed = rolling.NewRollingNumber()
}

func (m *poolMetrics) Monitor() {
	for u := range m.Updates {
		m.Mutex.RLock()

		m.Executed.Increment()
		m.MaxActiveRequests.UpdateMax(u.activeCount)

		m.Mutex.RUnlock()
	}
}
