package hystrix

import "time"
import "errors"

type Command struct {
	Run RunFunc
	Fallback FallbackFunc
	ResultChannel chan Result
	FallbackChannel chan Result
	ExecutorPoolName string
	ExecutorPoolSize int
}

func (command *Command) Execute() (Result) {
	future := command.Queue()
	return future.Value()
}

func (command *Command) Queue() (Future) {
	future := Future{ ValueChannel: make(chan Result) }
	go command.try_run(future.ValueChannel)
	return future
}

func (command *Command) try_run(value_channel chan Result) {
	// TODO: fallback if circuit is open
	var result Result
	result_channel := make(chan Result, 1)
	fallback_channel := make(chan Result, 1)

	command.ResultChannel = result_channel
	command.FallbackChannel = fallback_channel
	command.create_executor_pool()

	var executor *Executor

	select {
	case executor = <-executor_pools[command.ExecutorPoolName]:
		go executor.Run(command)

		defer func() {
			executor_pools[command.ExecutorPoolName] <- executor
		}()

		select {
		case result = <-result_channel:
			if result.Error != nil {
				// fallback if run fails
				value_channel <- command.try_fallback(result.Error)
			} else {
				value_channel <- result
			}	
		case <-time.After(time.Millisecond * 100): // TODO: make timeout dynamic
			// fallback if timeout is reached
			value_channel <- command.try_fallback(errors.New("Timeout"))
		}
	default:
		// fallback if executor pool is full
		value_channel <- command.try_fallback(errors.New("Executor Pool Full"))
	}
}

func (command *Command) try_fallback(err error) (Result) {
	go command.Fallback(err, command.FallbackChannel)
	// TODO: implement case for if fallback never returns
	return <-command.FallbackChannel
}

func (command *Command) create_executor_pool() {
	// TODO: handle concurrent calls to this to prevent races
	if command.ExecutorPoolName == "" {
		command.ExecutorPoolName = "hystrix"
	}
	if executor_pools[command.ExecutorPoolName] == nil {
		if command.ExecutorPoolSize == 0 {
			command.ExecutorPoolSize = 10
		}
		executor_pools[command.ExecutorPoolName] = make(chan *Executor, command.ExecutorPoolSize)
		for i := 0; i < command.ExecutorPoolSize; i++ {
			executor_pools[command.ExecutorPoolName] <- &Executor{}
		}
	}
}