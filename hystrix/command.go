package hystrix

import "time"
import "errors"

type Command struct {
	Run             RunFunc
	Fallback        FallbackFunc
	ResultChannel   chan Result
	FallbackChannel chan Result
	ExecutorPool    *ExecutorPool

	Observer ObserverFunc
}

func NewCommand(run RunFunc, fallback FallbackFunc) *Command {
	command := new(Command)

	command.Run = run
	command.Fallback = fallback
	command.ResultChannel = make(chan Result, 1)
	command.FallbackChannel = make(chan Result, 1)
	command.ExecutorPool = NewExecutorPool("hystrix", 10)

	return command
}

func (command *Command) Execute() Result {
	future := command.Queue()
	return future.Value()
}

func (command *Command) Queue() Future {
	future := Future{ValueChannel: make(chan Result)}
	go command.tryRun(future.ValueChannel)
	return future
}

func (command *Command) Observe() Observable {
	observable := Observable{Observer: command.Observer, ValueChannel: make(chan Result, 10)}
	go func() {
		for {
			value := <-observable.ValueChannel
			go observable.Observer(value)
		}
	}()
	go command.tryObserve(observable.ValueChannel)
	return observable
}

// TODO: figure out a way to merge try_run and try_observe

func (command *Command) tryRun(value_channel chan Result) {
	if command.ExecutorPool.Circuit.IsOpen() {
		// fallback if circuit is open due to too many recent failures
		value_channel <- command.tryFallback(errors.New("Circuit Open"))
	} else {
		select {
		case executor := <-command.ExecutorPool.Executors:
			defer func() {
				command.ExecutorPool.Executors <- executor
			}()

			go executor.Run(command)

			select {
			case result := <-command.ResultChannel:
				if result.Error != nil {
					// fallback if run fails
					value_channel <- command.tryFallback(result.Error)
				} else {
					value_channel <- result
				}
			case <-time.After(time.Millisecond * 100): // TODO: make timeout dynamic
				// fallback if timeout is reached
				value_channel <- command.tryFallback(errors.New("Timeout"))
			}
		default:
			// fallback if executor pool is full
			value_channel <- command.tryFallback(errors.New("Executor Pool Full"))
		}
	}
}

func (command *Command) tryFallback(err error) Result {
	if command.Fallback != nil {
		go command.Fallback(err, command.FallbackChannel)
		// TODO: implement case for if fallback never returns
		return <-command.FallbackChannel
	} else {
		return Result{Error: err}
	}
}

func (command *Command) tryObserve(value_channel chan Result) {
	if command.ExecutorPool.Circuit.IsOpen() {
		// fallback if circuit is open due to too many recent failures
		value_channel <- command.tryFallback(errors.New("Circuit Open"))
	} else {
		select {
		case executor := <-command.ExecutorPool.Executors:
			defer func() {
				command.ExecutorPool.Executors <- executor
			}()

			go executor.Run(command)

			for {
				select {
				case result, more := <-command.ResultChannel:
					if !more {
						return
					}
					if result.Error != nil {
						// fallback if run fails
						value_channel <- command.tryFallback(result.Error)
					} else {
						value_channel <- result
					}
				case <-time.After(time.Millisecond * 100): // TODO: make timeout dynamic
					// fallback if timeout is reached
					value_channel <- command.tryFallback(errors.New("Timeout"))
				}
			}
		default:
			// fallback if executor pool is full
			value_channel <- command.tryFallback(errors.New("Executor Pool Full"))
		}
	}
}
