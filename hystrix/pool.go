package hystrix

type ExecutorPool struct {
	Name    string
	Metrics *PoolMetrics
	Max     int
	Tickets chan *Ticket
}

func NewExecutorPool(name string) *ExecutorPool {
	p := &ExecutorPool{}
	p.Name = name
	p.Metrics = NewPoolMetrics(name)
	p.Max = GetConcurrency(name)

	p.Tickets = make(chan *Ticket, p.Max)
	for i := 0; i < p.Max; i++ {
		p.Tickets <- &Ticket{}
	}

	return p
}

func (p *ExecutorPool) Return(ticket *Ticket) {
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
