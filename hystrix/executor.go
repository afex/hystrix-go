package hystrix

type Executor struct {}

var executor_pools = make(map[string]*ExecutorPool)

func (executor *Executor) Run(command *Command) {
	command.Run(command.ResultChannel)
}

type ExecutorPool struct {
	Name string
	Size int
	Executors chan *Executor
	Circuit *CircuitBreaker
}

func NewExecutorPool(name string, size int) *ExecutorPool {
	// TODO: handle concurrent calls to this to prevent races

	if executor_pools[name] == nil {
		pool := &ExecutorPool{ 
			Name: name, 
			Size: size, 
			Executors: make(chan *Executor, size),
			Circuit: NewCircuitBreaker(),
		}
		for i := 0; i < pool.Size; i++ {
			pool.Executors <- &Executor{}
		}

		executor_pools[pool.Name] = pool
	}

	return executor_pools[name]
}

type CircuitBreaker struct {
	IsOpen bool
}

func NewCircuitBreaker() *CircuitBreaker {
	c := &CircuitBreaker{}
	c.IsOpen = false
	return c
}