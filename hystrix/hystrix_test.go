package hystrix

import (
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSuccess(t *testing.T) {
	defer FlushMetrics()

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
	defer FlushMetrics()

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
	defer FlushMetrics()
	ConfigureCommand("timeout", CommandConfig{Timeout: 100})

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
	defer FlushMetrics()
	ConfigureCommand("timeout", CommandConfig{Timeout: 100})

	resultChan := make(chan int)
	errChan := Go("timeout", func() error {
		time.Sleep(30 * time.Second)
		resultChan <- 1
		return nil
	}, nil)

	select {
	case _ = <-resultChan:
		t.Errorf("Should not get an response from resultChan")
	case _ = <-errChan:
	}
}

func TestMaxConcurrent(t *testing.T) {
	defer FlushMetrics()
	ConfigureCommand("max_concurrent", CommandConfig{MaxConcurrentRequests: 2})
	resultChan := make(chan int)

	fallback := func(err error) error {
		if err.Error() == "max concurrency" {
			resultChan <- 2
		}
		return nil
	}

	errChan1 := Go("max_concurrent", func() error {
		time.Sleep(1 * time.Second)
		return nil
	}, fallback)

	errChan2 := Go("max_concurrent", func() error {
		time.Sleep(1 * time.Second)
		resultChan <- 1
		return nil
	}, fallback)

	errChan3 := Go("max_concurrent", func() error {
		resultChan <- 1
		return nil
	}, fallback)

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
	defer FlushMetrics()
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
	defer FlushMetrics()
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

func TestCloseCircuitAfterSuccess(t *testing.T) {
	defer FlushMetrics()

	cb, _, err := GetCircuit("close_after_success")
	if err != nil {
		t.Fatalf("cant get circuit")
	}

	cb.SetOpen()
	if !cb.IsOpen() {
		t.Fatalf("circuit should be open")
	}

	time.Sleep(6 * time.Second)

	done := make(chan bool)
	errChan := Go("close_after_success", func() error {
		done <- true
		return nil
	}, nil)

	select {
	case _ = <-done:
		// do nothing
	case err := <-errChan:
		t.Fatal(err)
	}

	if cb.IsOpen() {
		t.Fatalf("circuit should be closed")
	}
}

func TestCloseErrorChannel(t *testing.T) {
	defer FlushMetrics()

	errChan := Go("close_channel", func() error {
		return nil
	}, nil)

	select {
	case _ = <-time.After(1 * time.Second):
		t.Fatal("timer fired before error channel was closed")
	case err := <-errChan:
		// errChan should be closed when command finishes
		if err != nil {
			t.Fatal("expected nil error")
		}
	}
}

func TestFailAfterTimeout(t *testing.T) {
	defer FlushMetrics()
	ConfigureCommand("fail_after_timeout", CommandConfig{Timeout: 10})

	errChan := Go("fail_after_timeout", func() error {
		time.Sleep(50 * time.Millisecond)
		return fmt.Errorf("foo")
	}, nil)

	select {
	case err := <-errChan:
		if err.Error() != "timeout" {
			t.Fatal("did not timeout as expected")
		}
	}

	// wait for command to fail, should not panic
	time.Sleep(100 * time.Millisecond)
}

func TestFallbackAfterRejected(t *testing.T) {
	Convey("with a circuit whose pool is full", t, func() {
		defer FlushMetrics()
		ConfigureCommand("fallback_after_rejected", CommandConfig{MaxConcurrentRequests: 1})
		cb, _, err := GetCircuit("fallback_after_rejected")
		if err != nil {
			t.Fatal(err)
		}
		<-cb.ExecutorPool.Tickets

		Convey("executing a successful fallback function due to rejection", func() {
			runChan := make(chan bool, 1)
			fallbackChan := make(chan bool, 1)
			errChan := Go("fallback_after_rejected", func() error {
				// if run executes after fallback, this will panic due to sending to a closed channel
				runChan <- true
				close(fallbackChan)
				return nil
			}, func(err error) error {
				fallbackChan <- true
				close(runChan)
				return nil
			})

			Convey("should not execute the run function", func() {
				err := <-errChan
				if err != nil {
					t.Fatal(err)
				}

				b := <-fallbackChan
				if b == false {
					t.Fatal("run function executed when it shouldn't have")
				}

				b = <-runChan
				if b == true {
					t.Fatal("run function executed when it shouldn't have")
				}
			})
		})
	})
}