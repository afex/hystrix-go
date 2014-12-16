// Package hystrix is a latency and fault tolerance library designed to isolate
// points of access to remote systems, services and 3rd party libraries, stop
// cascading failure and enable resilience in complex distributed systems where
// failure is inevitable.
//
// Based on the java project of the same name, by Netflix. https://github.com/Netflix/Hystrix
package hystrix

import (
	"errors"
	"fmt"
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
	errChan := make(chan error, 1)
	finished := make(chan bool, 1)

	// dont have methods with explicit params and returns
	// let data come in and out naturally, like with any closure
	// explicit error return to give place for us to kill switch the operation (fallback)

	circuit, err := GetCircuit(name)
	if err != nil {
		errChan <- err
	}

	go func() {
		defer func() { finished <- true }()

		tickets, err := ConcurrentThrottle(name)
		if err != nil {
			circuit.Health.Updates <- false
			errChan <- err
			return
		}

		// Circuits get opened when recent executions have shown to have a high error rate.
		// Rejecting new executions allows backends to recover, and the circuit will allow
		// new traffic when it feels a healthly state has returned.
		if circuit.IsOpen() {
			err := tryFallback(fallback, errors.New("circuit open"))
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
		select {
		case ticket := <-tickets:
			defer func() { tickets <- ticket }()
		default:
			circuit.Health.Updates <- false
			err := tryFallback(fallback, errors.New("max concurrency"))
			if err != nil {
				errChan <- err
				return
			}
		}

		runErr := run()
		if runErr != nil {
			circuit.Health.Updates <- false
			if fallback != nil {
				err := tryFallback(fallback, runErr)
				if err != nil {
					errChan <- err
				}
			} else {
				errChan <- runErr
			}
		}

		circuit.Health.Updates <- true
		circuit.Metrics.Updates <- &ExecutionMetric{Type: "success"}
	}()

	go func() {
		select {
		case <-finished:
		case <-time.After(GetTimeout(name)):
			circuit.Health.Updates <- false
			err := tryFallback(fallback, errors.New("timeout"))
			if err != nil {
				errChan <- err
			}
		}
	}()

	return errChan
}

func tryFallback(fallback fallbackFunc, err error) error {
	if fallback == nil {
		return nil
	}

	fallbackErr := fallback(err)
	if fallbackErr != nil {
		return fmt.Errorf("fallback failed with '%v'. run error was '%v'", fallbackErr, err)
	}

	return nil
}
