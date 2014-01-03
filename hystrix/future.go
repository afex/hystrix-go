package hystrix

type Future struct {
	isCached     bool
	cachedValue  Result
	ValueChannel chan Result
}

func (future *Future) Value() Result {
	// TODO: make value cached, since future only returns one value
	if !future.isCached {
		future.cachedValue = <-future.ValueChannel
		future.isCached = true
	}
	return future.cachedValue
}

type Observable struct {
	Observer     ObserverFunc
	ValueChannel chan Result
}
