package hystrix

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type runFunc func() error
type fallbackFunc func(error) error

// Go runs your function while tracking the health of previous calls to it.
// If your function begins slowing down or failing repeatedly, we will block
// new calls to it for you to give the dependent service time to repair.
//
// Define a fallback function if you want to define some code to execute during outages.
func Go(name string, run runFunc, fallback fallbackFunc) chan error {
	stop := false
	stopMutex := &sync.Mutex{}
	var ticket *struct{}
	ticketMutex := &sync.Mutex{}

	start := time.Now()

	errChan := make(chan error, 1)
	finished := make(chan bool, 1)

	// dont have methods with explicit params and returns
	// let data come in and out naturally, like with any closure
	// explicit error return to give place for us to kill switch the operation (fallback)

	circuit, _, err := GetCircuit(name)
	if err != nil {
		errChan <- err
		return errChan
	}

	go func() {
		defer func() { finished <- true }()

		// Circuits get opened when recent executions have shown to have a high error rate.
		// Rejecting new executions allows backends to recover, and the circuit will allow
		// new traffic when it feels a healthly state has returned.
		if !circuit.AllowRequest() {
			circuit.ReportEvent("short-circuit", start, 0)
			err := tryFallback(circuit, start, 0, fallback, errors.New("circuit open"))
			if err != nil {
				errChan <- err
			}
			return
		}

		// As backends falter, requests take longer but don't always fail.
		//
		// When requests slow down but the incoming rate of requests stays the same, you have to
		// run more at a time to keep up. By controlling concurrency during these situations, you can
		// shed load which accumulates due to the increasing ratio of active commands to incoming requests.
		ticketMutex.Lock()
		select {
		case ticket = <-circuit.executorPool.Tickets:
			ticketMutex.Unlock()
		default:
			ticketMutex.Unlock()
			circuit.ReportEvent("rejected", start, 0)
			err := tryFallback(circuit, start, 0, fallback, errors.New("max concurrency"))
			if err != nil {
				errChan <- err
			}
			return
		}

		runStart := time.Now()
		runErr := run()
		runDuration := time.Now().Sub(runStart)
		if runErr != nil {
			circuit.ReportEvent("failure", start, runDuration)
			err := tryFallback(circuit, start, runDuration, fallback, runErr)
			if err != nil {
				stopMutex.Lock()
				defer stopMutex.Unlock()
				if !stop {
					errChan <- err
				}
				return
			}
		}

		stopMutex.Lock()
		defer stopMutex.Unlock()
		if !stop {
			circuit.ReportEvent("success", start, runDuration)
		}
	}()

	go func() {
		defer func() {
			stopMutex.Lock()
			stop = true
			stopMutex.Unlock()

			ticketMutex.Lock()
			circuit.executorPool.Return(ticket)
			ticketMutex.Unlock()
		}()

		timer := time.NewTimer(getSettings(name).Timeout)
		defer timer.Stop()

		select {
		case <-finished:
		case <-timer.C:
			circuit.ReportEvent("timeout", start, 0)

			err := tryFallback(circuit, start, 0, fallback, errors.New("timeout"))
			if err != nil {
				errChan <- err
			}
		}
	}()

	return errChan
}

func tryFallback(circuit *CircuitBreaker, start time.Time, runDuration time.Duration, fallback fallbackFunc, err error) error {
	if fallback == nil {
		// If we don't have a fallback return the original error.
		return err
	}

	fallbackErr := fallback(err)
	if fallbackErr != nil {
		circuit.ReportEvent("fallback-failure", start, runDuration)
		return fmt.Errorf("fallback failed with '%v'. run error was '%v'", fallbackErr, err)
	}

	circuit.ReportEvent("fallback-success", start, runDuration)

	return nil
}
