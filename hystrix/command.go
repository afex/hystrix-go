package hystrix

import (
	"sync"
	"time"
	"fmt"
)

// command models the state used for a single execution on a circuit. "hystrix command" is commonly
// used to describe the pairing of your run/fallback functions with a circuit.
type command struct {
	sync.Mutex

	ticket       *struct{}
	start        time.Time
	errChan      chan error
	finished     chan bool
	fallbackOnce *sync.Once
	circuit      *CircuitBreaker
	run          runFunc
	fallback     fallbackFunc
	runDuration  time.Duration
	events       []string
	timedOut     bool
}


func (c *command) reportEvent(eventType string) {
	c.Lock()
	defer c.Unlock()

	c.events = append(c.events, eventType)
}

func (c *command) isTimedOut() bool {
	c.Lock()
	defer c.Unlock()

	return c.timedOut
}

// errorWithFallback triggers the fallback while reporting the appropriate metric events.
// If called multiple times for a single command, only the first will execute to insure
// accurate metrics and prevent the fallback from executing more than once.
func (c *command) errorWithFallback(err error) {
	c.fallbackOnce.Do(func() {
		eventType := "failure"
		if err == ErrCircuitOpen {
			eventType = "short-circuit"
		} else if err == ErrMaxConcurrency {
			eventType = "rejected"
		} else if err == ErrTimeout {
			eventType = "timeout"
		}

		c.reportEvent(eventType)
		fallbackErr := c.tryFallback(err)
		if fallbackErr != nil {
			c.errChan <- fallbackErr
		}
	})
}

func (c *command) tryFallback(err error) error {
	if c.fallback == nil {
		// If we don't have a fallback return the original error.
		return err
	}

	fallbackErr := c.fallback(err)
	if fallbackErr != nil {
		c.reportEvent("fallback-failure")
		return fmt.Errorf("fallback failed with '%v'. run error was '%v'", fallbackErr, err)
	}

	c.reportEvent("fallback-success")

	return nil
}
