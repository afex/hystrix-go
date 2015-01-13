package hystrix

import (
	"sync"
	"time"
)

// CircuitBreaker is created for each ExecutorPool to track whether requests
// should be attempted, or rejected if the Health of the circuit is too low.
type CircuitBreaker struct {
	Name      string
	Metrics   *Metrics
	Open      bool
	ForceOpen bool
	Mutex     *sync.RWMutex
}

var circuitBreakers map[string]*CircuitBreaker

func init() {
	circuitBreakers = make(map[string]*CircuitBreaker)
}

// GetCircuit returns the circuit for the given command
func GetCircuit(name string) (*CircuitBreaker, error) {
	_, ok := circuitBreakers[name]
	if !ok {
		circuitBreakers[name] = NewCircuitBreaker(name)
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
func NewCircuitBreaker(name string) *CircuitBreaker {
	c := &CircuitBreaker{}
	c.Name = name
	c.Metrics = NewMetrics()
	c.Mutex = &sync.RWMutex{}

	go c.watchHealth()

	return c
}

// watchHealth checks every second to see if it should toggle the
// open/closed state of the circuit
func (circuit *CircuitBreaker) watchHealth() {
	for {
		time.Sleep(1 * time.Second)
		circuit.toggleOpenFromMetrics(time.Now())
	}
}

// toggleOpenFromHealth updates the Open state based on a query to Health over
// the previous time window
func (circuit *CircuitBreaker) toggleOpenFromMetrics(now time.Time) {
	circuit.Mutex.Lock()
	defer circuit.Mutex.Unlock()

	healthy := circuit.Metrics.IsHealthy(now)
	if healthy && circuit.Open {
		circuit.Open = false
	} else if !healthy && !circuit.Open {
		circuit.Open = true
	}
}

// IsOpen is called before any Command execution to check whether or
// not it should be attempted. An "open" circuit means it is disabled.
func (circuit *CircuitBreaker) IsOpen() bool {
	circuit.Mutex.RLock()
	defer circuit.Mutex.RUnlock()
	
	return circuit.ForceOpen || circuit.Open
}

func (circuit *CircuitBreaker) AllowRequest() bool {
	return !circuit.IsOpen() || circuit.allowSingleTest()
}

func (circuit *CircuitBreaker) allowSingleTest() bool {
	return false
}
