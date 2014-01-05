package hystrix

type Executor struct{}

// TODO: this global sucks, refactor so state is not carried over between tests
var executorPools = make(map[string]*ExecutorPool)

func (executor *Executor) Run(command *Command) {
	command.Run(command.ResultChannel)
}

type ExecutorPool struct {
	Name      string
	Size      int
	Executors chan *Executor
	Circuit   *CircuitBreaker
}

func NewExecutorPool(name string, size int) *ExecutorPool {
	// TODO: handle concurrent calls to this to prevent races

	if executorPools[name] == nil {
		pool := &ExecutorPool{
			Name:      name,
			Size:      size,
			Executors: make(chan *Executor, size),
			Circuit:   NewCircuitBreaker(),
		}
		for i := 0; i < pool.Size; i++ {
			pool.Executors <- &Executor{}
		}

		executorPools[pool.Name] = pool
	}

	return executorPools[name]
}
