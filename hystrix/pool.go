package hystrix

import "sync"

var (
	poolMutex *sync.RWMutex
	pools     map[string]*executorPool
)

func init() {
	poolMutex = &sync.RWMutex{}
	pools = make(map[string]*executorPool)
}

type executorPool struct {
	Name    string
	Metrics *poolMetrics
	Max     int
	Tickets chan *struct{}
}

// GetCircuit returns the circuit for the given command and whether this call created it.
func getOrCreateExecutePool(name string) *executorPool {
	poolMutex.RLock()
	_, ok := pools[name]
	if !ok {
		poolMutex.RUnlock()
		poolMutex.Lock()
		defer poolMutex.Unlock()
		pools[name] = newExecutorPool(name)
	} else {
		defer poolMutex.RUnlock()
	}

	return pools[name]
}

func newExecutorPool(name string) *executorPool {
	p := &executorPool{}
	p.Name = name
	p.Metrics = newPoolMetrics(name)
	p.Max = getSettings(name).MaxConcurrentRequests

	p.Tickets = make(chan *struct{}, p.Max)
	for i := 0; i < p.Max; i++ {
		p.Tickets <- &struct{}{}
	}

	return p
}

func (p *executorPool) Return(ticket *struct{}) {
	if ticket == nil {
		return
	}

	p.Metrics.Updates <- poolMetricsUpdate{
		activeCount: p.ActiveCount(),
	}
	p.Tickets <- ticket
}

func (p *executorPool) ActiveCount() int {
	return p.Max - len(p.Tickets)
}
