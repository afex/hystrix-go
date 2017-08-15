package hystrix

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// CircuitBreaker is created for each ExecutorPool to track whether requests
// should be attempted, or rejected if the Health of the circuit is too low.
type CircuitBreaker struct {
	Name                   string
	open                   bool
	forceOpen              bool
	mutex                  *sync.RWMutex
	openedOrLastTestedTime int64

	executorPool *executorPool
	metrics      *metricExchange
	settings     *SettingsCollection
}

// GetCircuit returns the circuit for the given command and whether this call created it.
func (c *Circuits) GetCircuit(name string) (*CircuitBreaker, bool, error) {
	c.BreakersMutex.RLock()
	_, ok := c.Breakers[name]
	if !ok {
		c.BreakersMutex.RUnlock()
		c.BreakersMutex.Lock()
		defer c.BreakersMutex.Unlock()
		// because we released the rlock before we obtained the exclusive lock,
		// we need to double check that some other thread didn't beat us to
		// creation.
		if cb, ok := c.Breakers[name]; ok {
			return cb, false, nil
		}
		c.Breakers[name] = newCircuitBreaker(name, c.Settings)
	} else {
		defer c.BreakersMutex.RUnlock()
	}

	return c.Breakers[name], !ok, nil
}

// Flush purges all circuit and metric information from memory.
func (c *Circuits) Flush() {
	c.BreakersMutex.Lock()
	defer c.BreakersMutex.Unlock()

	for name, cb := range c.Breakers {
		cb.metrics.Reset()
		cb.executorPool.Metrics.Reset()
		delete(c.Breakers, name)
	}
}

// newCircuitBreaker creates a CircuitBreaker with associated Health
func newCircuitBreaker(name string, settingsCollection *SettingsCollection) *CircuitBreaker {
	c := &CircuitBreaker{}
	c.Name = name
	c.metrics = newMetricExchange(name, settingsCollection)
	c.executorPool = newExecutorPool(name, settingsCollection.getSettings(name).MaxConcurrentRequests)
	c.mutex = &sync.RWMutex{}
	c.settings = settingsCollection

	return c
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

	if uint64(circuit.metrics.Requests().Sum(time.Now())) < circuit.getSettings().RequestVolumeThreshold {
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
	if circuit.open && now > openedOrLastTestedTime+circuit.getSettings().SleepWindow.Nanoseconds() {
		swapped := atomic.CompareAndSwapInt64(&circuit.openedOrLastTestedTime, openedOrLastTestedTime, now)
		if swapped {
			log.Printf("hystrix-go: allowing single test to possibly close circuit %v", circuit.Name)
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

	log.Printf("hystrix-go: opening circuit %v", circuit.Name)

	circuit.openedOrLastTestedTime = time.Now().UnixNano()
	circuit.open = true
}

func (circuit *CircuitBreaker) setClose() {
	circuit.mutex.Lock()
	defer circuit.mutex.Unlock()

	if !circuit.open {
		return
	}

	log.Printf("hystrix-go: closing circuit %v", circuit.Name)

	circuit.open = false
	circuit.metrics.Reset()
}

func (circuit *CircuitBreaker) getSettings() *Settings {
	return circuit.settings.getSettings(circuit.Name)
}

// ReportEvent records command metrics for tracking recent error rates and exposing data to the dashboard.
func (circuit *CircuitBreaker) ReportEvent(eventTypes []string, start time.Time, runDuration time.Duration) error {
	if len(eventTypes) == 0 {
		return fmt.Errorf("no event types sent for metrics")
	}

	circuit.mutex.RLock()
	o := circuit.open
	circuit.mutex.RUnlock()
	if eventTypes[0] == "success" && o {
		circuit.setClose()
	}

	select {
	case circuit.metrics.Updates <- &commandExecution{
		Types:       eventTypes,
		Start:       start,
		RunDuration: runDuration,
	}:
	default:
		return CircuitError{Message: fmt.Sprintf("metrics channel (%v) is at capacity", circuit.Name)}
	}

	return nil
}
