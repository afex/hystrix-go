package hystrix

type Future struct {
	ValueChannel chan Result
}

func (future *Future) Value() (Result) {
	return <-future.ValueChannel
}