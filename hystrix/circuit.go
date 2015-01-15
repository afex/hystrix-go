package hystrix

import (
	"sync"
	"sync/atomic"
	"time"
)

// CircuitBreaker is created for each ExecutorPool to track whether requests
// should be attempted, or rejected if the Health of the circuit is too low.
type CircuitBreaker struct {
	Name                   string
	Metrics                *Metrics
	Open                   bool
	ForceOpen              bool
	Mutex                  *sync.RWMutex
	OpenedOrLastTestedTime int64
}

var (
	circuitBreakersMutex *sync.RWMutex
	circuitBreakers      map[string]*CircuitBreaker
)

func init() {
	circuitBreakersMutex = &sync.RWMutex{}
	circuitBreakers = make(map[string]*CircuitBreaker)
}

// GetCircuit returns the circuit for the given command
func GetCircuit(name string) (*CircuitBreaker, error) {
	circuitBreakersMutex.RLock()
	_, ok := circuitBreakers[name]
	if !ok {
		circuitBreakersMutex.RUnlock()
		circuitBreakersMutex.Lock()
		defer circuitBreakersMutex.Unlock()
		circuitBreakers[name] = NewCircuitBreaker(name)
	} else {
		defer circuitBreakersMutex.RUnlock()
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

	return c
}

// IsOpen is called before any Command execution to check whether or
// not it should be attempted. An "open" circuit means it is disabled.
func (circuit *CircuitBreaker) IsOpen() bool {
	circuit.Mutex.RLock()
	o := circuit.ForceOpen || circuit.Open
	circuit.Mutex.RUnlock()

	if o {
		return true
	}

	// TODO: configurable request volume threshold
	if circuit.Metrics.Requests.Sum(time.Now()) < 20 {
		return false
	}

	if !circuit.Metrics.IsHealthy(time.Now()) {
		// too many failures, open the circuit
		circuit.SetOpen()
		return true
	}

	return false
}

func (circuit *CircuitBreaker) AllowRequest() bool {
	return !circuit.IsOpen() || circuit.allowSingleTest()
}

func (circuit *CircuitBreaker) allowSingleTest() bool {
	circuit.Mutex.RLock()
	defer circuit.Mutex.RUnlock()

	now := time.Now().UnixNano()
	// TODO: configurable sleep window
	if circuit.Open && now > circuit.OpenedOrLastTestedTime+time.Duration(5*time.Second).Nanoseconds() {
		return atomic.CompareAndSwapInt64(&circuit.OpenedOrLastTestedTime, circuit.OpenedOrLastTestedTime, now)
	}

	return false
}

func (circuit *CircuitBreaker) SetOpen() {
	circuit.Mutex.Lock()
	defer circuit.Mutex.Unlock()

	circuit.OpenedOrLastTestedTime = time.Now().UnixNano()
	circuit.Open = true
}

func (circuit *CircuitBreaker) SetClose() {
	circuit.Mutex.Lock()
	defer circuit.Mutex.Unlock()

	circuit.Open = false
	circuit.Metrics.Reset()
}
