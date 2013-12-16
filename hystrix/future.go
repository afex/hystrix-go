package hystrix

type Future struct {
	ValueChannel chan Result
}

func (future *Future) Value() Result {
	// TODO: make value cached, since future only returns one value
	defer func() {
		close(future.ValueChannel)
	}()
	return <-future.ValueChannel
}

type Observable struct {
	Observer     ObserverFunc
	ValueChannel chan Result
}
