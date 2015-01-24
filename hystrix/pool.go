package hystrix

type ExecutorPool struct {
	Name    string
	Metrics *PoolMetrics
	Tickets chan *Ticket
}

func NewExecutorPool(name string) *ExecutorPool {
	p := &ExecutorPool{}
	p.Name = name
	p.Metrics = NewPoolMetrics(name)

	max := GetConcurrency(name)
	p.Tickets = make(chan *Ticket, max)
	for i := 0; i < max; i++ {
		p.Tickets <- &Ticket{}
	}

	return p
}

func (p *ExecutorPool) Return(ticket *Ticket) {
	p.Metrics.Updates <- struct{}{}
	p.Tickets <- ticket
}
