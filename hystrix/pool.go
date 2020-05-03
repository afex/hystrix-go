package hystrix

import "context"

type executorPool struct {
	Name    string
	Metrics *poolMetrics
	Max     int
	Tickets chan *struct{}
}

func newExecutorPool(ctx context.Context,name string) *executorPool {
	p := &executorPool{}
	p.Name = name
	p.Metrics = newPoolMetrics(ctx, name)
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
