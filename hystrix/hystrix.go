// Hystrix is a latency and fault tolerance library designed to isolate points of access to remote systems, services and 3rd party libraries, stop cascading failure and enable resilience in complex distributed systems where failure is inevitable.
//
// Based on the java project of the same name, by Netflix. https://github.com/Netflix/Hystrix
package hystrix

type RunFunc func(chan Result)
type FallbackFunc func(error, chan Result)
type ObserverFunc func(Result)

type Result struct {
	Result interface{}
	Error  error
}

func Execute(run RunFunc, fallback FallbackFunc) Result {
	command := NewCommand(run, fallback)
	return command.Execute()
}

func Queue(run RunFunc, fallback FallbackFunc) Future {
	command := NewCommand(run, fallback)
	return command.Queue()
}

func Observe(run RunFunc, fallback FallbackFunc, observer ObserverFunc) Observable {
	command := NewCommand(run, fallback)
	command.Observer = observer
	return command.Observe()
}
