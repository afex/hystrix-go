package hystrix

import "sync"

type PoolMetrics struct {
	Mutex *sync.RWMutex

	Name           string
	ActiveRequests *RollingNumber
	NumExecuted    *RollingNumber
}

func NewPoolMetrics(name string) *PoolMetrics {
	m := &PoolMetrics{}
	m.Name = name
	m.Mutex = &sync.RWMutex{}

	m.Reset()

	return m
}

func (m *PoolMetrics) Reset() {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	m.ActiveRequests = NewRollingNumber()
	m.NumExecuted = NewRollingNumber()
}
