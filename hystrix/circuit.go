package hystrix

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// CircuitBreakerInterface is the interface of circuit breaker
type CircuitBreakerInterface interface {
	// Name returns circuit name
	Name() string
	// AllowRequest returns true if circuit allow request
	AllowRequest() bool
	// IsOpen returns true if circuit is open
	IsOpen() bool
	// ForceOpen returns true if circuit being forced opened
	ForceOpen() bool
	// ReportEvent reports events to circuit metrics
	ReportEvent(eventTypes []string, start time.Time, runDuration time.Duration) error
	// Metrics returns circuit metrics
	Metrics() *metricExchange
	// ExecutorPool returns circuit executor pool
	ExecutorPool() ExecutorPoolInterface
}

// NoOpCircuitBreaker is created to allow all requests in all conditions
// Only enable metrics
type NoOpCircuitBreaker struct {
	name         string
	executorPool *noOpExecutorPool
	metrics      *metricExchange
}

// newNoOpCircuitBreaker creates a NoOpCircuitBreaker
func newNoOpCircuitBreaker(name string) *NoOpCircuitBreaker {
	c := &NoOpCircuitBreaker{}
	c.name = name
	c.metrics = newMetricExchange(name)
	c.executorPool = newNoOpExecutorPool(name)
	return c
}

// AllowRequest always returns true for NoOpCircuitBreaker
func (circuit *NoOpCircuitBreaker) AllowRequest() bool {
	return true
}

// IsOpen always returns false for NoOpCircuitBreaker
func (circuit *NoOpCircuitBreaker) IsOpen() bool {
	return false
}

// ForceOpen always returns false as can't force open NoOpCircuitBreaker
func (circuit *NoOpCircuitBreaker) ForceOpen() bool {
	return false
}

func (circuit *NoOpCircuitBreaker) Name() string {
	return circuit.name
}

func (circuit *NoOpCircuitBreaker) Metrics() *metricExchange {
	return circuit.metrics
}

func (circuit *NoOpCircuitBreaker) ReportEvent(eventTypes []string, start time.Time, runDuration time.Duration) error {
	if len(eventTypes) == 0 {
		return fmt.Errorf("no event types sent for metrics")
	}

	circuit.metrics.Updates <- &commandExecution{
		Types:       eventTypes,
		Start:       start,
		RunDuration: runDuration,
	}

	return nil
}

// ExecutorPool returns circuit executor pool
// NoOpCircuitBreaker will return noOpExecutorPool
func (circuit *NoOpCircuitBreaker) ExecutorPool() ExecutorPoolInterface {
	return circuit.executorPool
}

// CircuitBreaker is created for each ExecutorPool to track whether requests
// should be attempted, or rejected if the Health of the circuit is too low.
type CircuitBreaker struct {
	name                   string
	open                   bool
	forceOpen              bool
	mutex                  *sync.RWMutex
	openedOrLastTestedTime int64

	executorPool *executorPool
	metrics      *metricExchange
}

var (
	circuitBreakersMutex *sync.RWMutex
	circuitBreakers      map[string]CircuitBreakerInterface
)

func init() {
	circuitBreakersMutex = &sync.RWMutex{}
	circuitBreakers = make(map[string]CircuitBreakerInterface)
}

// GetCircuit returns the circuit for the given command and whether this call created it.
func GetCircuit(name string) (CircuitBreakerInterface, bool, error) {
	circuitBreakersMutex.RLock()
	_, ok := circuitBreakers[name]
	if !ok {
		circuitBreakersMutex.RUnlock()
		circuitBreakersMutex.Lock()
		defer circuitBreakersMutex.Unlock()
		// because we released the rlock before we obtained the exclusive lock,
		// we need to double check that some other thread didn't beat us to
		// creation.
		if cb, ok := circuitBreakers[name]; ok {
			return cb, false, nil
		}
		if getSettings(name).CircuitBreakerDisabled {
			circuitBreakers[name] = newNoOpCircuitBreaker(name)
		} else {
			circuitBreakers[name] = newCircuitBreaker(name)
		}
	} else {
		defer circuitBreakersMutex.RUnlock()
	}

	return circuitBreakers[name], !ok, nil
}

// Flush purges all circuit and metric information from memory.
func Flush() {
	circuitBreakersMutex.Lock()
	defer circuitBreakersMutex.Unlock()

	for name, cb := range circuitBreakers {
		cb.Metrics().Reset()
		cb.ExecutorPool().Metrics().Reset()
		delete(circuitBreakers, name)
	}
}

// newCircuitBreaker creates a CircuitBreaker with associated Health
func newCircuitBreaker(name string) *CircuitBreaker {
	c := &CircuitBreaker{}
	c.name = name
	c.metrics = newMetricExchange(name)
	c.executorPool = newExecutorPool(name)
	c.mutex = &sync.RWMutex{}
	return c
}

// toggleForceOpen allows manually causing the fallback logic for all instances
// of a given command.
func (circuit *CircuitBreaker) toggleForceOpen(toggle bool) error {
	newCircuit, _, err := GetCircuit(circuit.name)
	if err != nil {
		return err
	}

	if cb, ok := newCircuit.(*CircuitBreaker); ok {
		circuit = cb
		circuit.forceOpen = toggle
	}
	return nil
}

// ForceOpen returns if circuit had been force opened
func (circuit *CircuitBreaker) ForceOpen() bool {
	return circuit.forceOpen
}

// IsOpen is called before any Command execution to check whether or
// not it should be attempted. An "open" circuit means it is disabled.
func (circuit *CircuitBreaker) IsOpen() bool {
	circuit.mutex.RLock()
	o := circuit.forceOpen || circuit.open
	circuit.mutex.RUnlock()

	if o {
		return true
	}

	if uint64(circuit.metrics.Requests().Sum(time.Now())) < getSettings(circuit.name).RequestVolumeThreshold {
		return false
	}

	if !circuit.metrics.IsHealthy(time.Now()) {
		// too many failures, open the circuit
		circuit.setOpen()
		return true
	}

	return false
}

// AllowRequest is checked before a command executes, ensuring that circuit state and metric health allow it.
// When the circuit is open, this call will occasionally return true to measure whether the external service
// has recovered.
func (circuit *CircuitBreaker) AllowRequest() bool {
	return !circuit.IsOpen() || circuit.allowSingleTest()
}

func (circuit *CircuitBreaker) allowSingleTest() bool {
	circuit.mutex.RLock()
	defer circuit.mutex.RUnlock()

	now := time.Now().UnixNano()
	openedOrLastTestedTime := atomic.LoadInt64(&circuit.openedOrLastTestedTime)
	if circuit.open && now > openedOrLastTestedTime+getSettings(circuit.name).SleepWindow.Nanoseconds() {
		swapped := atomic.CompareAndSwapInt64(&circuit.openedOrLastTestedTime, openedOrLastTestedTime, now)
		if swapped {
			log.Printf("hystrix-go: allowing single test to possibly close circuit %v", circuit.name)
		}
		return swapped
	}

	return false
}

func (circuit *CircuitBreaker) setOpen() {
	circuit.mutex.Lock()
	defer circuit.mutex.Unlock()

	if circuit.open {
		return
	}

	log.Printf("hystrix-go: opening circuit %v", circuit.name)

	circuit.openedOrLastTestedTime = time.Now().UnixNano()
	circuit.open = true
}

func (circuit *CircuitBreaker) setClose() {
	circuit.mutex.Lock()
	defer circuit.mutex.Unlock()

	if !circuit.open {
		return
	}

	log.Printf("hystrix-go: closing circuit %v", circuit.name)

	circuit.open = false
	circuit.metrics.Reset()
}

// ReportEvent records command metrics for tracking recent error rates and exposing data to the dashboard.
func (circuit *CircuitBreaker) ReportEvent(eventTypes []string, start time.Time, runDuration time.Duration) error {
	if len(eventTypes) == 0 {
		return fmt.Errorf("no event types sent for metrics")
	}

	if eventTypes[0] == "success" && circuit.IsOpen() {
		circuit.setClose()
	}

	circuit.metrics.Updates <- &commandExecution{
		Types:       eventTypes,
		Start:       start,
		RunDuration: runDuration,
	}

	return nil
}

func (circuit *CircuitBreaker) Name() string {
	return circuit.name
}

func (circuit *CircuitBreaker) Metrics() *metricExchange {
	return circuit.metrics
}

func (circuit *CircuitBreaker) ExecutorPool() ExecutorPoolInterface {
	return circuit.executorPool
}
