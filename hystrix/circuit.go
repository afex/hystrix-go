package hystrix

// CircuitBreaker is created for each ExecutorPool to track whether requests
// should be attempted, or rejected if the Health of the circuit is too low.
type CircuitBreaker struct {
	health    Health
	ForceOpen bool
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

func (circuit *CircuitBreaker) AllowRequest() bool {
	return !circuit.IsOpen() || circuit.allowSingleTest()
}

func (circuit *CircuitBreaker) allowSingleTest() bool {
	return false
}

func (circuit *CircuitBreaker) markSuccess() {

}
