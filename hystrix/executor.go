package hystrix

// Executor represents an available slot for concurrent commands to run.
type Executor struct{}

// TODO: this global sucks, refactor so state is not carried over between tests
var executorPools = make(map[string]*ExecutorPool)

// Run is used to ensure that commands only execute when an executor is available.
func (executor *Executor) Run(command *Command) {
	command.Runner.Run(command.ResultChannel)
}

// ExecutorPool provides a channel for easy checkout/checkin of Executors
type ExecutorPool struct {
	Name      string
	Size      int
	Executors chan *Executor
	Circuit   *CircuitBreaker
}

// NewExecutorPool creates a new pool given a name and number of executors to instantiate.
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
