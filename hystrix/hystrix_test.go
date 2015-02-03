package hystrix

import (
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSuccess(t *testing.T) {
	Convey("with a command which sends to a channel", t, func() {
		defer Flush()

		resultChan := make(chan int)
		errChan := Go("", func() error {
			resultChan <- 1
			return nil
		}, nil)

		Convey("reading from that channel should provide the expected value", func() {
			So(<-resultChan, ShouldEqual, 1)
		})
		Convey("no errors should be returned", func() {
			So(len(errChan), ShouldEqual, 0)
		})
	})
}

func TestFallback(t *testing.T) {
	Convey("with a command which fails, and whose fallback sends to a channel", t, func() {
		defer Flush()

		resultChan := make(chan int)
		errChan := Go("", func() error {
			return fmt.Errorf("error")
		}, func(err error) error {
			if err.Error() == "error" {
				resultChan <- 1
			}
			return nil
		})

		Convey("reading from that channel should provide the expected value", func() {
			So(<-resultChan, ShouldEqual, 1)
		})
		Convey("no errors should be returned", func() {
			So(len(errChan), ShouldEqual, 0)
		})
	})
}

func TestTimeout(t *testing.T) {
	Convey("with a command which times out, and whose fallback sends to a channel", t, func() {
		defer Flush()
		ConfigureCommand("", CommandConfig{Timeout: 100})

		resultChan := make(chan int)
		errChan := Go("", func() error {
			time.Sleep(1 * time.Second)
			resultChan <- 1
			return nil
		}, func(err error) error {
			if err.Error() == "timeout" {
				resultChan <- 2
			}
			return nil
		})

		Convey("reading from that channel should provide the expected value", func() {
			So(<-resultChan, ShouldEqual, 2)
		})
		Convey("no errors should be returned", func() {
			So(len(errChan), ShouldEqual, 0)
		})
	})
}

func TestTimeoutEmptyFallback(t *testing.T) {
	Convey("with a command which times out, and has no fallback", t, func() {
		defer Flush()
		ConfigureCommand("", CommandConfig{Timeout: 100})

		resultChan := make(chan int)
		errChan := Go("", func() error {
			time.Sleep(1 * time.Second)
			resultChan <- 1
			return nil
		}, nil)

		Convey("a timeout error should be returned", func() {
			So((<-errChan).Error(), ShouldEqual, "timeout")
		})
	})
}

func TestMaxConcurrent(t *testing.T) {
	Convey("if a command has max concurrency set to 2", t, func() {
		defer Flush()
		ConfigureCommand("", CommandConfig{MaxConcurrentRequests: 2})
		resultChan := make(chan int)

		run := func() error {
			time.Sleep(1 * time.Second)
			resultChan <- 1
			return nil
		}

		Convey("and 3 of those commands try to execute at the same time", func() {
			var good, bad int

			for i := 0; i < 3; i++ {
				errChan := Go("", run, nil)
				time.Sleep(10 * time.Millisecond)

				select {
				case err := <-errChan:
					if err.Error() == "max concurrency" {
						bad++
					}
				default:
					good++
				}
			}

			Convey("one will return a 'max concurrency' error", func() {
				So(bad, ShouldEqual, 1)
				So(good, ShouldEqual, 2)
			})
		})
	})
}

func TestForceOpenCircuit(t *testing.T) {
	Convey("when a command with a forced open circuit is run", t, func() {
		defer Flush()

		cb, _, err := GetCircuit("")
		So(err, ShouldEqual, nil)

		cb.toggleForceOpen(true)

		errChan := Go("", func() error {
			return nil
		}, nil)

		Convey("a 'circuit open' error is returned", func() {
			So((<-errChan).Error(), ShouldEqual, "circuit open")
		})
	})
}

func TestNilFallbackRunError(t *testing.T) {
	Convey("when your run function returns an error and you have no fallback", t, func() {
		defer Flush()
		errChan := Go("", func() error {
			return fmt.Errorf("run_error")
		}, nil)

		Convey("the returned error should be the run error", func() {
			err := <-errChan

			So(err.Error(), ShouldEqual, "run_error")
		})
	})
}

func TestFailedFallback(t *testing.T) {
	Convey("when your run and fallback functions return an error", t, func() {
		defer Flush()
		errChan := Go("", func() error {
			return fmt.Errorf("run_error")
		}, func(err error) error {
			return fmt.Errorf("fallback_error")
		})

		Convey("the returned error should contain both", func() {
			err := <-errChan

			So(err.Error(), ShouldEqual, "fallback failed with 'fallback_error'. run error was 'run_error'")
		})
	})
}

func TestCloseCircuitAfterSuccess(t *testing.T) {
	Convey("when a circuit is open", t, func() {
		defer Flush()
		cb, _, err := GetCircuit("")
		So(err, ShouldEqual, nil)

		cb.setOpen()

		Convey("commands immediately following should short-circuit", func() {
			errChan := Go("", func() error {
				return nil
			}, nil)

			So((<-errChan).Error(), ShouldEqual, "circuit open")
		})

		Convey("and a successful command is run after the sleep window", func() {
			time.Sleep(6 * time.Second)

			done := make(chan bool, 1)
			Go("", func() error {
				done <- true
				return nil
			}, nil)

			Convey("the circuit should be closed", func() {
				So(<-done, ShouldEqual, true)
				So(cb.isOpen(), ShouldEqual, false)
			})
		})
	})
}

func TestFailAfterTimeout(t *testing.T) {
	Convey("when a slow command fails after the timeout fires", t, func() {
		defer Flush()
		ConfigureCommand("", CommandConfig{Timeout: 10})

		errChan := Go("", func() error {
			time.Sleep(50 * time.Millisecond)
			return fmt.Errorf("foo")
		}, nil)

		Convey("we do not panic", func() {
			So((<-errChan).Error(), ShouldEqual, "timeout")
			// wait for command to fail, should not panic
			time.Sleep(100 * time.Millisecond)
		})
	})
}

func TestFallbackAfterRejected(t *testing.T) {
	Convey("with a circuit whose pool is full", t, func() {
		defer Flush()
		ConfigureCommand("", CommandConfig{MaxConcurrentRequests: 1})
		cb, _, err := GetCircuit("")
		if err != nil {
			t.Fatal(err)
		}
		<-cb.executorPool.Tickets

		Convey("executing a successful fallback function due to rejection", func() {
			runChan := make(chan bool, 1)
			fallbackChan := make(chan bool, 1)
			Go("", func() error {
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
				So(<-fallbackChan, ShouldBeTrue)
				So(<-runChan, ShouldBeFalse)
			})
		})
	})
}

func TestReturnTicket(t *testing.T) {
	Convey("with a run command that doesn't return", t, func() {
		defer Flush()

		ConfigureCommand("", CommandConfig{Timeout: 10})

		errChan := Go("", func() error {
			c := make(chan struct{})
			<-c // should block
			return nil
		}, nil)

		Convey("the ticket returns to the pool after the timeout", func() {
			err := <-errChan
			So(err.Error(), ShouldEqual, "timeout")

			cb, _, err := GetCircuit("")
			So(err, ShouldBeNil)
			So(cb.executorPool.ActiveCount(), ShouldEqual, 0)
		})
	})
}
