package hystrix

import "sync"

type PoolMetrics struct {
	Mutex   *sync.RWMutex
	Updates chan struct{}

	Name              string
	MaxActiveRequests *RollingNumber
	Executed          *RollingNumber
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

	m.MaxActiveRequests = NewRollingNumber()
	m.Executed = NewRollingNumber()
}

func (m *PoolMetrics) Monitor() {
	for _ = range m.Updates {
		m.Executed.Increment()
	}
}
