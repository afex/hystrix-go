package hystrix

import (
	"sync"
)

type bufferedExecutorPool struct {
	Name                        string
	Metrics                     *bufferedPoolMetrics
	Max                         int
	MaxQueueSize                int
	QueueSizeRejectionThreshold int
	TicketAvailableChan         chan *struct{}
	WaitingTicket               chan *struct{}
	Tickets                     chan *struct{}

	mutex sync.Mutex
}

func newBufferedExecutorPool(name string) *bufferedExecutorPool {
	p := &bufferedExecutorPool{}
	p.Name = name
	p.mutex = sync.Mutex{}
	p.Metrics = newBufferedPoolMetrics(name)
	p.Max = getSettings(name).MaxConcurrentRequests
	p.QueueSizeRejectionThreshold = getSettings(name).QueueSizeRejectionThreshold
	p.WaitingTicket = make(chan *struct{}, p.QueueSizeRejectionThreshold)

	p.Tickets = make(chan *struct{}, p.Max)
	for i := 0; i < p.Max; i++ {
		p.Tickets <- &struct{}{}
	}
	for i := 0; i < p.QueueSizeRejectionThreshold; i++ {
		p.WaitingTicket <- &struct{}{}
	}

	return p
}

func (p *bufferedExecutorPool) Return(ticket *struct{}) {
	if ticket == nil {
		return
	}

	p.Metrics.Updates <- bufferedPoolMetricsUpdate{
		activeCount:  p.ActiveCount(),
		waitingCount: p.WaitingCount(),
	}
	p.Tickets <- ticket
}

func (p *bufferedExecutorPool) ReturnWaitingTicket(ticket *struct{}) {
	if ticket == nil {
		return
	}

	p.WaitingTicket <- ticket
}

func (p *bufferedExecutorPool) ActiveCount() int {
	return p.Max - len(p.Tickets)
}

func (p *bufferedExecutorPool) WaitingCount() int {
	return p.QueueSizeRejectionThreshold - len(p.WaitingTicket)
}
