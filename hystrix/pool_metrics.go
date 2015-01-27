package hystrix

import "sync"

type PoolMetrics struct {
	Mutex   *sync.RWMutex
	Updates chan struct{}

	Name              string
	MaxActiveRequests *rollingNumber
	Executed          *rollingNumber
}

func NewPoolMetrics(name string) *PoolMetrics {
	m := &PoolMetrics{}
	m.Name = name
	m.Updates = make(chan struct{})
	m.Mutex = &sync.RWMutex{}

	m.Reset()

	go m.Monitor()

	return m
}

func (m *PoolMetrics) Reset() {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	m.MaxActiveRequests = newRollingNumber()
	m.Executed = newRollingNumber()
}

func (m *PoolMetrics) Monitor() {
	for _ = range m.Updates {
		m.Executed.Increment()
	}
}
