package hystrix

// CircuitBreaker is created for each ExecutorPool to track whether requests
// should be attempted, or rejected if the Health of the circuit is too low.
type CircuitBreaker struct {
	health    *Health
	ForceOpen bool
}

// GetCircuitWithUpdater returns the circuit for the given command, and a write-only channel to 
// update the perceived health of a command.
//
// Enough failures over a duration and the Command will be treated as unhealthy, preventing new
// executions.
func GetCircuitWithUpdater(name string) (*CircuitBreaker, chan<- *healthUpdate, error) {
	return NewCircuitBreaker(), make(chan *healthUpdate), nil
}

// NewCircuitBreaker creates a CircuitBreaker with associated Health
func NewCircuitBreaker() *CircuitBreaker {
	c := &CircuitBreaker{}
	c.health = NewHealth()

	return c
}

// IsOpen is called before any Command execution to check whether or
// not it should be attempted. An "open" circuit means it is disabled.
func (circuit *CircuitBreaker) IsOpen() bool {
	return circuit.ForceOpen || !circuit.health.IsHealthy()
}
