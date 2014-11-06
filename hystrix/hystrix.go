// Package hystrix is a latency and fault tolerance library designed to isolate
// points of access to remote systems, services and 3rd party libraries, stop
// cascading failure and enable resilience in complex distributed systems where
// failure is inevitable.
//
// Based on the java project of the same name, by Netflix. https://github.com/Netflix/Hystrix
package hystrix

import (
	"errors"
	"time"
)

type hystrixFunc func() error

// Go runs your function while tracking the health of previous calls to it.
// If your function begins slowing down or failing repeatedly, we will block 
// new calls to it for you to give the dependent service time to repair.  
// 
// Define a fallback function if you want to define some code to execute during outages.
func Go(name string, run hystrixFunc, fallback hystrixFunc) chan error {
	errChan := make(chan error)
	finished := make(chan bool, 1)

	// dont have methods with explicit params and returns
	// let data come in and out naturally, like with any closure
	// explicit error return to give place for us to kill switch the operation (fallback)

	// TODO: check circuit breaker
	// TODO: throttle per command name

	go func() {
		err := run()

		if err != nil {
			if fallback != nil {
				err := fallback()
				if err != nil {
					// TODO: this should contain merged error information to show both run and fallback messages
					errChan <- err
				}
			}
			errChan <- err
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