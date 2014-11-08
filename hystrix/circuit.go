package hystrix

// CircuitBreaker is created for each ExecutorPool to track whether requests
// should be attempted, or rejected if the Health of the circuit is too low.
type CircuitBreaker struct {
	health    *Health
	Updates   chan *healthUpdate
	ForceOpen bool
}

var circuitBreakers map[string]*CircuitBreaker

func init() {
	circuitBreakers = make(map[string]*CircuitBreaker)
}

// GetCircuit returns the circuit for the given command
func GetCircuit(name string) (*CircuitBreaker, error) {
	_, ok := circuitBreakers[name]
	if !ok {
		circuitBreakers[name] = NewCircuitBreaker()
	}

	return circuitBreakers[name], nil
}

// ForceCircuitOpen allows manually causing the fallback logic for all instances
// of a given command.
func ForceCircuitOpen(name string, toggle bool) error {
	circuit, err := GetCircuit(name)
	if err != nil {
		return err
	}

	circuit.ForceOpen = toggle
	return nil
}

// NewCircuitBreaker creates a CircuitBreaker with associated Health
func NewCircuitBreaker() *CircuitBreaker {
	c := &CircuitBreaker{}
	c.health = NewHealth()
	c.Updates = make(chan *healthUpdate)
	return c
}

// IsOpen is called before any Command execution to check whether or
// not it should be attempted. An "open" circuit means it is disabled.
func (circuit *CircuitBreaker) IsOpen() bool {
	return circuit.ForceOpen || !circuit.health.IsHealthy()
}
