package hystrix

type ExecutorPool struct {
	Name string
	Metrics *PoolMetrics
}

func NewExecutorPool(name string) *ExecutorPool {
	p := &ExecutorPool{}
	p.Name = name
	p.Metrics = NewPoolMetrics(name)

	return p
}
