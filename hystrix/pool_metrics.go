package hystrix

import "sync"

type poolMetrics struct {
	Mutex   *sync.RWMutex
	Updates chan poolMetricsUpdate

	Name              string
	MaxActiveRequests *rollingNumber
	Executed          *rollingNumber
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

	m.MaxActiveRequests = newRollingNumber()
	m.Executed = newRollingNumber()
}

func (m *poolMetrics) Monitor() {
	for u := range m.Updates {
		m.Mutex.RLock()

		m.Executed.Increment()
		m.MaxActiveRequests.UpdateMax(u.activeCount)

		m.Mutex.RUnlock()
	}
}
