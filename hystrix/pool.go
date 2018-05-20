package hystrix

type executorPool struct {
	Name    string
	Metrics *poolMetrics
	Max     int
	Tickets chan *struct{}
	excessive chan *struct{}
}

func newExecutorPool(name string) *executorPool {
	p := &executorPool{}
	p.Name = name
	p.Metrics = newPoolMetrics(name)
	p.Max = getSettings(name).MaxConcurrentRequests

	p.Tickets = make(chan *struct{}, p.Max)
	p.excessive = make(chan *struct{}, 0)
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
	select {
	case _ = <-p.excessive:
		return
	default:
		p.Tickets <- ticket
	}
}

func (p *executorPool) ActiveCount() int {
	return p.Max - len(p.Tickets) + len(p.excessive)
}

func (p *executorPool) resize(n int) {
	if n < 1 {
		// dont resize to something < 1
		return
	}

	active := p.ActiveCount()

	alloc_cnt := n - active
	if alloc_cnt < 0 {
		excessive := alloc_cnt * -1
		// a channel that serves as a counter with excessive tickets
		p.excessive = make(chan *struct{}, excessive)
		for i := 0; i < excessive; i++ {
			p.excessive <- &struct{}{}
		}

	}

	p.Max = n
	p.Tickets = make(chan *struct{}, p.Max)

	for i := 0; i < alloc_cnt; i++ {
		p.Tickets <- &struct{}{}
	}
}