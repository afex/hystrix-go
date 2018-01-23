package hystrix

import (
	"fmt"
	"log"
	"sync"
	"time"
)

type runFunc func() error
type fallbackFunc func(error) error

// A CircuitError is an error which models various failure states of execution,
// such as the circuit being open or a timeout.
type CircuitError struct {
	Message string
}

func (e CircuitError) Error() string {
	return "hystrix: " + e.Message
}

// command models the state used for a single execution on a circuit. "hystrix command" is commonly
// used to describe the pairing of your run/fallback functions with a circuit.
type command struct {
	sync.Mutex

	name        string
	ticket      *struct{}
	start       time.Time
	errChan     chan error
	finished    chan bool
	circuit     *CircuitBreaker
	run         runFunc
	fallback    fallbackFunc
	runDuration time.Duration
	events      []string
}

var (
	// ErrMaxConcurrency occurs when too many of the same named command are executed at the same time.
	ErrMaxConcurrency = CircuitError{Message: "max concurrency"}
	// ErrCircuitOpen returns when an execution attempt "short circuits". This happens due to the circuit being measured as unhealthy.
	ErrCircuitOpen = CircuitError{Message: "circuit open"}
	// ErrTimeout occurs when the provided function takes too long to execute.
	ErrTimeout = CircuitError{Message: "timeout"}
)

// Go runs your function while tracking the health of previous calls to it.
// If your function begins slowing down or failing repeatedly, we will block
// new calls to it for you to give the dependent service time to repair.
//
// Define a fallback function if you want to define some code to execute during outages.
func Go(name string, run runFunc, fallback fallbackFunc) chan error {
	cmd := &command{
		name:     name,
		run:      run,
		fallback: fallback,
		start:    time.Now(),
		errChan:  make(chan error, 1),
		finished: make(chan bool, 1),
	}

	return cmd.Go()
}

// Do runs your function in a synchronous manner, blocking until either your function succeeds
// or an error is returned, including hystrix circuit errors
func Do(name string, run runFunc, fallback fallbackFunc) error {

	done := make(chan struct{}, 1)

	r := func() error {
		err := run()
		if err != nil {
			return err
		}

		done <- struct{}{}
		return nil
	}

	f := func(e error) error {
		err := fallback(e)
		if err != nil {
			return err
		}

		done <- struct{}{}
		return nil
	}

	cmd := &command{
		name:     name,
		run:      r,
		start:    time.Now(),
		errChan:  make(chan error, 1),
		finished: make(chan bool, 1),
	}
	if fallback != nil {
		cmd.fallback = f
	}

	errChan := cmd.Go()

	select {
	case <-done:
		return nil
	case err := <-errChan:
		return err
	}
}

func (c *command) Go() chan error {
	// dont have methods with explicit params and returns
	// let data come in and out naturally, like with any closure
	// explicit error return to give place for us to kill switch the operation (fallback)

	circuit, _, err := GetCircuit(c.name)
	if err != nil {
		c.errChan <- err
		return c.errChan
	}
	c.circuit = circuit
	ticketCond := sync.NewCond(c)
	ticketChecked := false
	// When the caller extracts error from returned errChan, it's assumed that
	// the ticket's been returned to executorPool. Therefore, returnTicket() can
	// not run after cmd.errorWithFallback().
	returnTicket := func() {
		c.Lock()
		// Avoid releasing before a ticket is acquired.
		for !ticketChecked {
			ticketCond.Wait()
		}
		c.circuit.executorPool.Return(c.ticket)
		c.Unlock()
	}
	// Shared by the following two goroutines. It ensures only the faster
	// goroutine runs errWithFallback() and reportAllEvent().
	returnOnce := &sync.Once{}
	reportAllEvent := func() {
		err := c.circuit.ReportEvent(c.events, c.start, c.runDuration)
		if err != nil {
			log.Print(err)
		}
	}

	go func() {
		defer func() { c.finished <- true }()

		// Circuits get opened when recent executions have shown to have a high error rate.
		// Rejecting new executions allows backends to recover, and the circuit will allow
		// new traffic when it feels a healthly state has returned.
		if !c.circuit.AllowRequest() {
			c.Lock()
			// It's safe for another goroutine to go ahead releasing a nil ticket.
			ticketChecked = true
			ticketCond.Signal()
			c.Unlock()
			returnOnce.Do(func() {
				returnTicket()
				c.errorWithFallback(ErrCircuitOpen)
				reportAllEvent()
			})
			return
		}

		// As backends falter, requests take longer but don't always fail.
		//
		// When requests slow down but the incoming rate of requests stays the same, you have to
		// run more at a time to keep up. By controlling concurrency during these situations, you can
		// shed load which accumulates due to the increasing ratio of active commands to incoming requests.
		c.Lock()
		select {
		case c.ticket = <-circuit.executorPool.Tickets:
			ticketChecked = true
			ticketCond.Signal()
			c.Unlock()
		default:
			ticketChecked = true
			ticketCond.Signal()
			c.Unlock()
			returnOnce.Do(func() {
				returnTicket()
				c.errorWithFallback(ErrMaxConcurrency)
				reportAllEvent()
			})
			return
		}

		runStart := time.Now()
		runErr := c.run()
		returnOnce.Do(func() {
			defer reportAllEvent()
			c.runDuration = time.Since(runStart)
			returnTicket()
			if runErr != nil {
				c.errorWithFallback(runErr)
				return
			}
			c.reportEvent("success")
		})
	}()

	go func() {
		timer := time.NewTimer(getSettings(c.name).Timeout)
		defer timer.Stop()

		select {
		case <-c.finished:
			// returnOnce has been executed in another goroutine
		case <-timer.C:
			returnOnce.Do(func() {
				returnTicket()
				c.errorWithFallback(ErrTimeout)
				reportAllEvent()
			})
			return
		}
	}()

	return c.errChan
}

func (c *command) reportEvent(eventType string) {
	c.Lock()
	defer c.Unlock()

	c.events = append(c.events, eventType)
}

// errorWithFallback triggers the fallback while reporting the appropriate metric events.
func (c *command) errorWithFallback(err error) {
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
