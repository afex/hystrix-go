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
	sync.RWMutex

	ticket         *struct{}
	overflowTicket *struct{}
	start          time.Time
	errChan        chan error
	finished       chan bool
	timeoutChan    chan struct{}
	fallbackOnce   *sync.Once
	circuit        *CircuitBreaker
	run            runFunc
	fallback       fallbackFunc
	runDuration    time.Duration
	events         []string
	timedOut       bool
	ticketChecked  chan struct{}
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
		run:           run,
		fallback:      fallback,
		start:         time.Now(),
		errChan:       make(chan error, 1),
		finished:      make(chan bool, 1),
		fallbackOnce:  &sync.Once{},
		timeoutChan:   make(chan struct{}, 1),
		ticketChecked: make(chan struct{}, 1),
	}

	// dont have methods with explicit params and returns
	// let data come in and out naturally, like with any closure
	// explicit error return to give place for us to kill switch the operation (fallback)

	circuit, _, err := GetCircuit(name)
	if err != nil {
		cmd.errChan <- err
		return cmd.errChan
	}
	cmd.circuit = circuit

	go func() {
		defer func() {
			cmd.finished <- true
		}()

		// Circuits get opened when recent executions have shown to have a high error rate.
		// Rejecting new executions allows backends to recover, and the circuit will allow
		// new traffic when it feels a healthly state has returned.
		if !cmd.circuit.AllowRequest() {
			cmd.errorWithFallback(ErrCircuitOpen)
			close(cmd.ticketChecked)
			return
		}

		// As backends falter, requests take longer but don't always fail.
		//
		// When requests slow down but the incoming rate of requests stays the same, you have to
		// run more at a time to keep up. By controlling concurrency during these situations, you can
		// shed load which accumulates due to the increasing ratio of active commands to incoming requests.

		select {
		case t := <-circuit.executorPool.Tickets:
			cmd.setTicket(t)

		default:
			select {
			case t := <-circuit.executorPool.WaitingTicket:
				cmd.reportEvent("queued")
				cmd.setOverflowTicket(t)
			default: // Unable to get execution or waiting ticket, error with MaxConcurrency
				cmd.errorWithFallback(ErrMaxConcurrency)
				close(cmd.ticketChecked)
				return
			}

			// Unable to execute the cmd but was able to get the waiting slot
			select {
			case executionTicket := <-circuit.executorPool.Tickets:
				// return the ticket right away as it is not required
				cmd.circuit.executorPool.ReturnWaitingTicket(cmd.overflowTicket)
				cmd.setTicket(executionTicket)
				isOpen := circuit.IsOpen()
				if isOpen {
					cmd.errorWithFallback(ErrCircuitOpen)
					close(cmd.ticketChecked)
					return
				}
			case <-cmd.timeoutChan:
				// return the ticket right away as it is not required
				cmd.circuit.executorPool.ReturnWaitingTicket(cmd.overflowTicket)
				close(cmd.ticketChecked)
				return
			}
		}

		close(cmd.ticketChecked)
		runStart := time.Now()
		runErr := run()

		if !cmd.isTimedOut() {
			cmd.setRunDuration(time.Since(runStart))

			if runErr != nil {
				cmd.errorWithFallback(runErr)
				return
			}

			cmd.reportEvent("success")
		}
	}()

	go func() {
		defer func() {
			<-cmd.ticketChecked
			cmd.Lock()
			cmd.circuit.executorPool.Return(cmd.ticket)
			copyEvents := append([]string(nil), cmd.events...)
			cmd.Unlock()

			err := cmd.circuit.ReportEvent(copyEvents, cmd.start, cmd.getRunDuration())
			if err != nil {
				log.Print(err)
			}
		}()

		timer := time.NewTimer(getSettings(name).Timeout)
		defer timer.Stop()

		select {
		case <-cmd.finished:
		case <-timer.C:
			close(cmd.timeoutChan)
			// mark as timeout only if the reason is timeout, if the job was in overflowQueue mark it as MaxConcurrency
			if cmd.getOverflowTicket() == nil {
				cmd.Lock()
				cmd.timedOut = true
				cmd.Unlock()
				cmd.errorWithFallback(ErrTimeout)
			} else {
				// even if the execution was waiting in queue and then timed-out while executing, mark it as ErrMaxConcurrency
				cmd.errorWithFallback(ErrMaxConcurrency)
			}
			return
		}
	}()

	return cmd.errChan
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

	var errChan chan error
	if fallback == nil {
		errChan = Go(name, r, nil)
	} else {
		errChan = Go(name, r, f)
	}

	select {
	case <-done:
		return nil
	case err := <-errChan:
		return err
	}
}

func (c *command) reportEvent(eventType string) {
	c.Lock()
	defer c.Unlock()

	c.events = append(c.events, eventType)
}

func (c *command) isTimedOut() bool {
	c.RLock()
	defer c.RUnlock()

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

func (c *command) getTicket() *struct{} {
	c.RLock()
	defer c.RUnlock()

	return c.ticket
}
func (c *command) setTicket(t *struct{}) {
	c.Lock()
	defer c.Unlock()

	c.ticket = t
}
func (c *command) getOverflowTicket() *struct{} {
	c.RLock()
	defer c.RUnlock()

	return c.overflowTicket
}

func (c *command) setOverflowTicket(t *struct{}) {
	c.Lock()
	defer c.Unlock()

	c.overflowTicket = t
}

func (c *command) setRunDuration(duration time.Duration) {
	c.Lock()
	defer c.Unlock()
	c.runDuration = duration
}

func (c *command) getRunDuration() time.Duration {
	c.RLock()
	defer c.RUnlock()
	return c.runDuration
}
