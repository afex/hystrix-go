package hystrix

import (
	"sync"

	"github.com/afex/hystrix-go/hystrix/rolling"
	"time"
)

type poolMetrics struct {
	Mutex             *sync.RWMutex
	Updates           chan poolMetricsUpdate

	Name              string
	MaxActiveRequests *rolling.Number
	Executed          *rolling.Number
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

	m.MaxActiveRequests = rolling.NewNumber()
	m.Executed = rolling.NewNumber()
}

func (m *poolMetrics) CloseUpdates() {
	close(m.Updates)
}

func (m *poolMetrics) GetMaxActiveRequests() float64 {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	return m.MaxActiveRequests.Max(time.Now())
}

func (m *poolMetrics) Monitor() {
	for u := range m.Updates {
		m.Mutex.RLock()

		m.Executed.Increment(1)
		m.MaxActiveRequests.UpdateMax(float64(u.activeCount))

		m.Mutex.RUnlock()
	}
}
