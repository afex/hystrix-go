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

type hystrixRunFunc func() error
type hystrixFallbackFunc func(error) error

// Go runs your function while tracking the health of previous calls to it.
// If your function begins slowing down or failing repeatedly, we will block
// new calls to it for you to give the dependent service time to repair.
//
// Define a fallback function if you want to define some code to execute during outages.
func Go(name string, run hystrixRunFunc, fallback hystrixFallbackFunc) chan error {
	errChan := make(chan error, 1)
	finished := make(chan bool, 1)

	// dont have methods with explicit params and returns
	// let data come in and out naturally, like with any closure
	// explicit error return to give place for us to kill switch the operation (fallback)

	// TODO: check circuit breaker
	// TODO: throttle per command name

	go func() {
		runErr := run()

		if runErr != nil {
			if fallback != nil {
				fallbackErr := fallback(runErr)
				if fallbackErr != nil {
					errChan <- fmt.Errorf("fallback failed with '%v'. run error was '%v'", fallbackErr, runErr)
				}
			} else {
				errChan <- runErr
			}
		}

		finished <- true
	}()

	go func() {
		select {
		case <-finished:
		case <-time.After(timeoutForCommand(name)):
			errChan <- errors.New("timeout")
		}
	}()

	return errChan
}
