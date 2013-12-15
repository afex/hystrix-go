package hystrix

type RunFunc func(chan Result)
type FallbackFunc func(error, chan Result)

type Result struct {
	Result interface{}
	Error error
}

func Execute(run RunFunc, fallback FallbackFunc) (Result) {
	command := Command{Run: run, Fallback: fallback}
	return command.Execute()
}

func Queue(run RunFunc, fallback FallbackFunc) (Future) {
	command := Command{Run: run, Fallback: fallback}
	return command.Queue()
}