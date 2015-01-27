package hystrix

type ExecutorPool struct {
	Name    string
	Metrics *PoolMetrics
	Max     int
	Tickets chan *struct{}
}

func NewExecutorPool(name string) *ExecutorPool {
	p := &ExecutorPool{}
	p.Name = name
	p.Metrics = NewPoolMetrics(name)
	p.Max = GetConcurrency(name)

	p.Tickets = make(chan *struct{}, p.Max)
	for i := 0; i < p.Max; i++ {
		p.Tickets <- &struct{}{}
	}

	return p
}

func (p *ExecutorPool) Return(ticket *struct{}) {
	if ticket == nil {
		return
	}

	p.Metrics.Updates <- struct{}{}
	p.Metrics.MaxActiveRequests.UpdateMax(p.ActiveCount())
	p.Tickets <- ticket
}

func (p *ExecutorPool) ActiveCount() int {
	return p.Max - len(p.Tickets)
}
