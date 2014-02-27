package hystrix

import "time"
import "errors"

// Command is the core struct for hystrix execution.  It represents both the
// happy path and fallback when accessing remote systems, as well as result delivery.
type Command struct {
	// TODO: better name than "inner"
	InnerCommand    CommandInterface
	ResultChannel   chan Result
	FallbackChannel chan Result
	ExecutorPool    *ExecutorPool
}

// TODO: refactor so "Command" is the interface not the struct
type CommandInterface interface {
	Run(chan Result)
	Fallback(error, chan Result)
}

// BUG(keith): all commands share the same executor pool, instead of being grouped by name

// NewCommand maps the given run and fallback functions with result channels and an executor pool
func NewCommand(inner CommandInterface) *Command {
	command := new(Command)

	command.InnerCommand = inner
	command.ResultChannel = make(chan Result, 1)
	command.FallbackChannel = make(chan Result, 1)
	command.ExecutorPool = NewExecutorPool("hystrix", 10)

	return command
}

// Execute runs the command synchronously, blocking until the result (or fallback) is returned
func (command *Command) Execute() Result {
	channel := command.Queue()
	return <-channel
}

// Queue runs the command asynchronously, immediately returning a channel which the result (or fallback) will be sent to.
func (command *Command) Queue() chan Result {
	channel := make(chan Result, 1)
	go command.tryRun(channel)
	return channel
}

func (command *Command) tryRun(valueChannel chan Result) {
	defer close(valueChannel)
	if command.ExecutorPool.Circuit.IsOpen() {
		// fallback if circuit is open due to too many recent failures
		valueChannel <- command.tryFallback(errors.New("circuit open"))
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
					valueChannel <- command.tryFallback(result.Error)
				} else {
					valueChannel <- result
				}
			case <-time.After(time.Millisecond * 100): // TODO: make timeout dynamic
				// fallback if timeout is reached
				valueChannel <- command.tryFallback(errors.New("timeout"))
			}
		default:
			// fallback if executor pool is full
			valueChannel <- command.tryFallback(errors.New("executor pool full"))
		}
	}
}

func (command *Command) tryFallback(err error) Result {
	go command.InnerCommand.Fallback(err, command.FallbackChannel)
	// TODO: implement case for if fallback never returns
	return <-command.FallbackChannel
}
