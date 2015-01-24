package hystrix

import (
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// CircuitBreaker is created for each ExecutorPool to track whether requests
// should be attempted, or rejected if the Health of the circuit is too low.
type CircuitBreaker struct {
	Name                   string
	Open                   bool
	ForceOpen              bool
	Mutex                  *sync.RWMutex
	OpenedOrLastTestedTime int64

	ExecutorPool *ExecutorPool
	Metrics      *Metrics
}

var (
	circuitBreakersMutex *sync.RWMutex
	circuitBreakers      map[string]*CircuitBreaker
)

func init() {
	circuitBreakersMutex = &sync.RWMutex{}
	circuitBreakers = make(map[string]*CircuitBreaker)
}

// GetCircuit returns the circuit for the given command and whether this call created it.
func GetCircuit(name string) (*CircuitBreaker, bool, error) {
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

	return circuitBreakers[name], !ok, nil
}

func FlushMetrics() {
	circuitBreakersMutex.Lock()
	defer circuitBreakersMutex.Unlock()

	for name, cb := range circuitBreakers {
		cb.Metrics.Reset()
		delete(circuitBreakers, name)
	}
}

// ForceCircuitOpen allows manually causing the fallback logic for all instances
// of a given command.
func ForceCircuitOpen(name string, toggle bool) error {
	circuit, _, err := GetCircuit(name)
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
	c.Metrics = NewMetrics(name)
	c.ExecutorPool = NewExecutorPool(name)
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

	if circuit.Metrics.Requests().Sum(time.Now()) < GetRequestVolumeThreshold(circuit.Name) {
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
	if circuit.Open && now > circuit.OpenedOrLastTestedTime+GetSleepWindow(circuit.Name).Nanoseconds() {
		swapped := atomic.CompareAndSwapInt64(&circuit.OpenedOrLastTestedTime, circuit.OpenedOrLastTestedTime, now)
		if swapped {
			log.Printf("hystrix-go: allowing single test to possibly close circuit %v", circuit.Name)
		}
		return swapped
	}

	return false
}

func (circuit *CircuitBreaker) SetOpen() {
	circuit.Mutex.Lock()
	defer circuit.Mutex.Unlock()

	log.Printf("hystrix-go: opening circuit %v", circuit.Name)

	circuit.OpenedOrLastTestedTime = time.Now().UnixNano()
	circuit.Open = true
}

func (circuit *CircuitBreaker) SetClose() {
	circuit.Mutex.Lock()
	defer circuit.Mutex.Unlock()

	log.Printf("hystrix-go: closing circuit %v", circuit.Name)

	circuit.Open = false
	circuit.Metrics.Reset()
}

func (circuit *CircuitBreaker) ReportEvent(eventType string, start time.Time, runDuration time.Duration) error {
	if eventType == "success" && circuit.IsOpen() {
		circuit.SetClose()
	}

	totalDuration := time.Now().Sub(start)

	circuit.Metrics.Updates <- &ExecutionMetric{
		Type:          eventType,
		Time:          time.Now(),
		RunDuration:   runDuration,
		TotalDuration: totalDuration,
	}

	return nil
}
