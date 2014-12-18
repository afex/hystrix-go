package hystrix

import (
	"fmt"
	"testing"
	"time"
)

func TestSuccess(t *testing.T) {
	resultChan := make(chan int)
	errChan := Go("good", func() error {
		resultChan <- 1
		return nil
	}, nil)

	select {
	case result := <-resultChan:
		if result != 1 {
			t.Errorf("result wasn't 1: %v", result)
		}
	case err := <-errChan:
		t.Errorf(err.Error())
	}
}

func TestFallback(t *testing.T) {
	resultChan := make(chan int)
	errChan := Go("bad", func() error {
		return fmt.Errorf("error")
	}, func(err error) error {
		if err.Error() == "error" {
			resultChan <- 1
		}
		return nil
	})

	select {
	case result := <-resultChan:
		if result != 1 {
			t.Errorf("result wasn't 1: %v", result)
		}
	case err := <-errChan:
		t.Errorf(err.Error())
	}
}

func TestTimeout(t *testing.T) {
	SetTimeout("timeout", time.Millisecond*100)

	resultChan := make(chan int)
	errChan := Go("timeout", func() error {
		time.Sleep(1 * time.Second)
		resultChan <- 1
		return nil
	}, func(err error) error {
		if err.Error() == "timeout" {
			resultChan <- 2
		}
		return nil
	})

	select {
	case result := <-resultChan:
		if result != 2 {
			t.Errorf("didn't get fallback value 2: %v", result)
		}
	case err := <-errChan:
		t.Errorf(err.Error())
	}
}

func TestTimeoutEmptyFallback(t *testing.T) {
	SetTimeout("timeout", time.Millisecond*100)

	resultChan := make(chan int)
	errChan := Go("timeout", func() error {
		time.Sleep(1 * time.Second)
		resultChan <- 1
		return nil
	}, nil)

	select {
	case _ = <-resultChan:
		t.Errorf("Should not get an response from resultChan")
	case _ = <-errChan:
	}
}

/*
func TestTimeoutFallback(t *testing.T) {
	SetTimeout("timeout", time.Millisecond*100)

	resultChan := make(chan int)
	errChan := Go("timeout", func() error {
		time.Sleep(1 * time.Second)
		resultChan <- 1
		return nil
	}, func(err error) error {
		time.Sleep(1 * time.Second)
		resultChan <- 1
		return nil
	})

	select {
	case _ = <-resultChan:
		t.Errorf("Should not get an response from resultChan")
	case _ = <-errChan:
	}
}
*/

// TODO: how can we be sure the fallback is triggered from full pool.  error type?
func TestMaxConcurrent(t *testing.T) {
	SetConcurrency("max_concurrent", 2)

	resultChan := make(chan int)
	errChan1 := Go("max_concurrent", func() error {
		time.Sleep(1 * time.Second)
		return nil
	}, nil)
	errChan2 := Go("max_concurrent", func() error {
		time.Sleep(1 * time.Second)
		resultChan <- 1
		return nil
	}, nil)
	errChan3 := Go("max_concurrent", func() error {
		resultChan <- 1
		return nil
	}, func(err error) error {
		if err.Error() == "max concurrency" {
			resultChan <- 2
		}
		return nil
	})

	select {
	case result := <-resultChan:
		if result != 2 {
			t.Errorf("didn't get fallback value 2: %v", result)
		}
	case err := <-errChan1:
		t.Errorf(err.Error())
	case err := <-errChan2:
		t.Errorf(err.Error())
	case err := <-errChan3:
		t.Errorf(err.Error())
	}
}

func TestOpenCircuit(t *testing.T) {
	ForceCircuitOpen("open_circuit", true)

	resultChan := make(chan int)
	errChan := Go("open_circuit", func() error {
		resultChan <- 2
		return nil
	}, func(err error) error {
		if err.Error() == "circuit open" {
			resultChan <- 1
		}
		return nil
	})

	select {
	case result := <-resultChan:
		if result != 1 {
			t.Errorf("didn't get fallback 1: %v", result)
		}
	case err := <-errChan:
		t.Errorf(err.Error())
	}
}

func TestFailedFallback(t *testing.T) {
	errChan := Go("fallback_error", func() error {
		return fmt.Errorf("run_error")
	}, func(err error) error {
		return fmt.Errorf("fallback_error")
	})

	err := <-errChan

	if err.Error() != "fallback failed with 'fallback_error'. run error was 'run_error'" {
		t.Errorf("did not get expected error: %v", err)
	}
}
