package hystrix

import (
	"errors"
	"time"
)

// Command is the core struct for hystrix execution.  It maps the user-defined
// Runner with channels for delivering results.
type Command struct {
	Runner       Runner
	ExecutorPool *ExecutorPool
}

// Runner is the user-defined methods for the execution/fallback
// of the command, as well as configurable settings.
type Runner interface {
	Run() (interface{}, error)
	Fallback(error) (interface{}, error)
	PoolName() string
	Timeout() time.Duration
}

// NewCommand maps the given run and fallback functions with result channels and an executor pool
func NewCommand(runner Runner) *Command {
	command := new(Command)

	command.Runner = runner
	command.ExecutorPool = NewExecutorPool(runner.PoolName(), 10)

	return command
}

// Execute runs the command synchronously, blocking until the result (or fallback) is returned
func (command *Command) Execute() (interface{}, error) {
	results, errors := command.Queue()
	select {
	case result := <-results:
		return result, nil
	case err := <-errors:
		return nil, err
	}
}

// Queue runs the command asynchronously, immediately returning a channel which the result (or fallback) will be sent to.
func (command *Command) Queue() (chan interface{}, chan error) {
	results := make(chan interface{}, 1)
	errors := make(chan error, 1)
	go command.tryRun(results, errors)
	return results, errors
}

func (command *Command) tryRun(valueChannel chan interface{}, errorChannel chan error) {
	defer close(valueChannel)
	if command.ExecutorPool.Circuit.IsOpen() {
		// fallback if circuit is open due to too many recent failures
		result, err := command.Runner.Fallback(errors.New("circuit open"))
		if err != nil {
			errorChannel <- err
		} else {
			valueChannel <- result
		}
	} else {
		select {
		case executor := <-command.ExecutorPool.Executors:
			defer func() {
				command.ExecutorPool.Executors <- executor
			}()

			innerRunChannel := make(chan interface{}, 1)
			innerErrorChannel := make(chan error, 1)
			go func() {
				result, err := executor.Run(command)
				if err != nil {
					innerErrorChannel <- err
				} else {
					innerRunChannel <- result
				}
			}()

			select {
			case result := <-innerRunChannel:
				valueChannel <- result
			case err := <-innerErrorChannel:
				// fallback if run fails
				command.tryFallback(err, valueChannel, errorChannel)
			case <-time.After(command.Runner.Timeout()):
				// fallback if timeout is reached
				command.tryFallback(errors.New("timeout"), valueChannel, errorChannel)
			}
		default:
			// fallback if executor pool is full
			command.tryFallback(errors.New("executor pool full"), valueChannel, errorChannel)
		}
	}
}

func (command *Command) tryFallback(triggeredError error, valueChannel chan interface{}, errorChannel chan error) {
	result, err := command.Runner.Fallback(triggeredError)
	if err != nil {
		errorChannel <- err
	} else {
		valueChannel <- result
	}
}
