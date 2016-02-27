package hystrix

import (
	"math"
	"sync/atomic"
)

// ExecutorPoolInterface is the interface of executor pool
type ExecutorPoolInterface interface {
	// Name returns pool name
	Name() string
	// Ticket returns a chan, client can get ticket from this chan
	Ticket() <-chan *struct{}
	// Return returns back the ticket client previously get from pool
	Return(ticket *struct{})
	// ActiveCount returns the active count of `clients` in the pool
	ActiveCount() int
	// Max returns the maximum tickets number in pool
	Max() int
	// Metrics returns metrics object associated with this pool
	Metrics() *poolMetrics
}

// noOpExecutorPool have infinite tickets, it will not block client's requests
type noOpExecutorPool struct {
	name    string
	count   int32
	metrics *poolMetrics
}

// newNoOpExecutorPool creates noOpExecutorPool
func newNoOpExecutorPool(name string) *noOpExecutorPool {
	p := &noOpExecutorPool{}
	p.name = name
	p.count = 0
	p.metrics = newPoolMetrics(name)
	return p
}

func (p *noOpExecutorPool) Name() string {
	return p.name
}

// noOpTicket is the ticket instance noOpExecutorPool gives to clients
var noOpTicket = &struct{}{}

// Ticket will always return a buffered chan with 1 noOpTicket in it
// It will close the chan so no more tickets being send to the chan
func (p *noOpExecutorPool) Ticket() <-chan *struct{} {
	atomic.AddInt32(&p.count, 1)

	ticket := make(chan *struct{}, 1)
	ticket <- noOpTicket
	// close means no more values will be send to this chan
	// it's safe to receive from close chan
	close(ticket)
	return ticket
}

// Return will update noOpExecutorPool's metrics
func (p *noOpExecutorPool) Return(ticket *struct{}) {
	if ticket == nil {
		return
	}

	p.metrics.Updates <- poolMetricsUpdate{
		activeCount: p.ActiveCount(),
	}
	atomic.AddInt32(&p.count, -1)
}

func (p *noOpExecutorPool) Metrics() *poolMetrics {
	return p.metrics
}

func (p *noOpExecutorPool) ActiveCount() int {
	return int(p.count)
}

// Max of noOpExecutorPool will return MaxInt32
func (p *noOpExecutorPool) Max() int {
	return math.MaxInt32
}

type executorPool struct {
	name    string
	metrics *poolMetrics
	max     int
	tickets chan *struct{}
}

func newExecutorPool(name string) *executorPool {
	p := &executorPool{}
	p.name = name
	p.metrics = newPoolMetrics(name)
	p.max = getSettings(name).MaxConcurrentRequests

	p.tickets = make(chan *struct{}, p.max)
	for i := 0; i < p.max; i++ {
		p.tickets <- &struct{}{}
	}

	return p
}

func (p *executorPool) Name() string {
	return p.name
}

func (p *executorPool) Ticket() <-chan *struct{} {
	return p.tickets
}

func (p *executorPool) Return(ticket *struct{}) {
	if ticket == nil {
		return
	}

	p.metrics.Updates <- poolMetricsUpdate{
		activeCount: p.ActiveCount(),
	}
	p.tickets <- ticket
}

func (p *executorPool) ActiveCount() int {
	return p.max - len(p.tickets)
}

func (p *executorPool) Max() int {
	return p.max
}

func (p *executorPool) Metrics() *poolMetrics {
	return p.metrics
}
