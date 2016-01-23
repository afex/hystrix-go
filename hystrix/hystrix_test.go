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

			Convey("no errors should be returned", func() {
				So(len(errChan), ShouldEqual, 0)
			})
			Convey("metrics are recorded", func() {
				time.Sleep(10 * time.Millisecond)
				cb, _, _ := GetCircuit("")
				So(cb.metrics.DefaultCollector().Successes().Sum(time.Now()), ShouldEqual, 1)
			})
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

			Convey("no errors should be returned", func() {
				So(len(errChan), ShouldEqual, 0)
			})
			Convey("metrics are recorded", func() {
				time.Sleep(10 * time.Millisecond)
				cb, _, _ := GetCircuit("")
				So(cb.metrics.DefaultCollector().Successes().Sum(time.Now()), ShouldEqual, 0)
				So(cb.metrics.DefaultCollector().Failures().Sum(time.Now()), ShouldEqual, 1)
				So(cb.metrics.DefaultCollector().FallbackSuccesses().Sum(time.Now()), ShouldEqual, 1)
			})
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
			if err == ErrTimeout {
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
			So(<-errChan, ShouldResemble, ErrTimeout)

			Convey("metrics are recorded", func() {
				time.Sleep(10 * time.Millisecond)
				cb, _, _ := GetCircuit("")
				So(cb.metrics.DefaultCollector().Successes().Sum(time.Now()), ShouldEqual, 0)
				So(cb.metrics.DefaultCollector().Timeouts().Sum(time.Now()), ShouldEqual, 1)
				So(cb.metrics.DefaultCollector().FallbackSuccesses().Sum(time.Now()), ShouldEqual, 0)
				So(cb.metrics.DefaultCollector().FallbackFailures().Sum(time.Now()), ShouldEqual, 0)
			})
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
					if err == ErrMaxConcurrency {
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

		cb.ToggleForceOpen(true)

		errChan := Go("", func() error {
			return nil
		}, nil)

		Convey("a 'circuit open' error is returned", func() {
			So(<-errChan, ShouldResemble, ErrCircuitOpen)

			Convey("metrics are recorded", func() {
				time.Sleep(10 * time.Millisecond)
				cb, _, _ := GetCircuit("")
				So(cb.metrics.DefaultCollector().Successes().Sum(time.Now()), ShouldEqual, 0)
				So(cb.metrics.DefaultCollector().ShortCircuits().Sum(time.Now()), ShouldEqual, 1)
			})
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

			So(<-errChan, ShouldResemble, ErrCircuitOpen)
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
				time.Sleep(10 * time.Millisecond)
				So(cb.IsOpen(), ShouldEqual, false)
			})
		})
	})
}

func TestFailAfterTimeout(t *testing.T) {
	Convey("when a slow command fails after the timeout fires", t, func() {
		defer Flush()
		ConfigureCommand("", CommandConfig{Timeout: 10})

		out := make(chan struct{}, 2)
		errChan := Go("", func() error {
			time.Sleep(50 * time.Millisecond)
			return fmt.Errorf("foo")
		}, func(err error) error {
			out <- struct{}{}
			return err
		})

		Convey("we do not panic", func() {
			So((<-errChan).Error(), ShouldContainSubstring, "timeout")
			// wait for command to fail, should not panic
			time.Sleep(100 * time.Millisecond)
		})

		Convey("we do not call the fallback twice", func() {
			time.Sleep(100 * time.Millisecond)
			So(len(out), ShouldEqual, 1)
		})
	})
}

func TestSlowFallbackOpenCircuit(t *testing.T) {
	Convey("with an open circuit and a slow fallback", t, func() {
		defer Flush()

		ConfigureCommand("", CommandConfig{Timeout: 10})

		cb, _, err := GetCircuit("")
		So(err, ShouldEqual, nil)
		cb.setOpen()

		out := make(chan struct{}, 2)

		Convey("when the command short circuits", func() {
			Go("", func() error {
				return nil
			}, func(err error) error {
				time.Sleep(100 * time.Millisecond)
				out <- struct{}{}
				return nil
			})

			Convey("the fallback only fires for the short-circuit, not both", func() {
				time.Sleep(250 * time.Millisecond)
				So(len(out), ShouldEqual, 1)

				Convey("and a timeout event is not recorded", func() {
					So(cb.metrics.DefaultCollector().ShortCircuits().Sum(time.Now()), ShouldEqual, 1)
					So(cb.metrics.DefaultCollector().Timeouts().Sum(time.Now()), ShouldEqual, 0)
				})
			})
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
			So(err, ShouldResemble, ErrTimeout)

			cb, _, err := GetCircuit("")
			So(err, ShouldBeNil)
			So(cb.executorPool.ActiveCount(), ShouldEqual, 0)
		})
	})
}

func TestDo(t *testing.T) {
	Convey("with a command which succeeds", t, func() {
		defer Flush()

		out := make(chan bool, 1)
		run := func() error {
			out <- true
			return nil
		}

		Convey("the run function is executed", func() {
			err := Do("", run, nil)
			So(err, ShouldBeNil)
			So(<-out, ShouldEqual, true)
		})
	})

	Convey("with a command which fails", t, func() {
		defer Flush()

		run := func() error {
			return fmt.Errorf("i failed")
		}

		Convey("with no fallback", func() {
			err := Do("", run, nil)
			Convey("the error is returned", func() {
				So(err.Error(), ShouldEqual, "i failed")
			})
		})

		Convey("with a succeeding fallback", func() {
			out := make(chan bool, 1)
			fallback := func(err error) error {
				out <- true
				return nil
			}

			err := Do("", run, fallback)

			Convey("the fallback is executed", func() {
				So(err, ShouldBeNil)
				So(<-out, ShouldEqual, true)
			})
		})

		Convey("with a failing fallback", func() {
			fallback := func(err error) error {
				return fmt.Errorf("fallback failed")
			}

			err := Do("", run, fallback)

			Convey("both errors are returned", func() {
				So(err.Error(), ShouldEqual, "fallback failed with 'fallback failed'. run error was 'i failed'")
			})
		})
	})

	Convey("with a command which times out", t, func() {
		defer Flush()

		ConfigureCommand("", CommandConfig{Timeout: 10})

		err := Do("", func() error {
			time.Sleep(100 * time.Millisecond)
			return nil
		}, nil)

		Convey("the timeout error is returned", func() {
			So(err.Error(), ShouldEqual, "hystrix: timeout")
		})
	})
}
