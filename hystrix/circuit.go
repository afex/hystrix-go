package hystrix

type CircuitBreaker struct {
	health    Health
	ForceOpen bool
}

func NewCircuitBreaker() *CircuitBreaker {
	c := &CircuitBreaker{}
	c.health = NewHealth()

	return c
}

func (circuit *CircuitBreaker) IsOpen() bool {
	return circuit.ForceOpen || !circuit.health.IsHealthy()
}