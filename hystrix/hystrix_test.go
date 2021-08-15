package hystrix

import (
	"context"
	"fmt"
	"testing"
	"time"

	"testing/quick"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSuccess(t *testing.T) {
	Convey("with a command which sends to a channel", t, func() {
		defer Flush()

		resultChan := make(chan int)
		errChan := GoC(context.Background(), "", func(ctx context.Context) error {
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
		errChan := GoC(context.Background(), "", func(ctx context.Context) error {
			return fmt.Errorf("error")
		}, func(ctx context.Context, err error) error {
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
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		errChan := GoC(ctx, "", func(ctx context.Context) error {
			time.Sleep(1 * time.Second)
			resultChan <- 1
			return nil
		}, func(ctx context.Context, err error) error {
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
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		errChan := GoC(ctx, "", func(ctx context.Context) error {
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

		run := func(ctx context.Context) error {
			time.Sleep(1 * time.Second)
			resultChan <- 1
			return nil
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		Convey("and 3 of those commands try to execute at the same time", func() {
			var good, bad int

			for i := 0; i < 3; i++ {
				errChan := GoC(ctx, "", run, nil)
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

		cb.toggleForceOpen(true)

		errChan := GoC(context.Background(), "", func(ctx context.Context) error {
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
		errChan := GoC(context.Background(), "", func(ctx context.Context) error {
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
		errChan := GoC(context.Background(), "", func(ctx context.Context) error {
			return fmt.Errorf("run_error")
		}, func(ctx context.Context, err error) error {
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
			errChan := GoC(context.Background(), "", func(ctx context.Context) error {
				return nil
			}, nil)

			So(<-errChan, ShouldResemble, ErrCircuitOpen)
		})

		Convey("and a successful command is run after the sleep window", func() {
			time.Sleep(6 * time.Second)

			done := make(chan bool, 1)
			GoC(context.Background(), "", func(ctx context.Context) error {
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
		errChan := GoC(context.Background(), "", func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			return fmt.Errorf("foo")
		}, func(ctx context.Context, err error) error {
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
			GoC(context.Background(), "", func(ctx context.Context) error {
				return nil
			}, func(ctx context.Context, err error) error {
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
			GoC(context.Background(), "", func(ctx context.Context) error {
				// if run executes after fallback, this will panic due to sending to a closed channel
				runChan <- true
				close(fallbackChan)
				return nil
			}, func(ctx context.Context, err error) error {
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

func TestReturnTicket_QuickCheck(t *testing.T) {
	compareTicket := func() bool {
		defer Flush()
		ConfigureCommand("", CommandConfig{Timeout: 2})

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		errChan := GoC(ctx, "", func(ctx context.Context) error {
			//there are multiple ways to block here, the following sequence of steps
			//will block:: c := make(chan struct{});  <-c; return nil // should block
			//however, this would leak the internal GoC.func goroutine
			//another non-leaking way to do this would be to simply: return ErrTimeout
			c := make(chan struct{})
			<-c // should block (hence we add an exception in go-leak)
			return nil
		}, nil)

		err := <-errChan
		So(err, ShouldResemble, ErrTimeout)
		cb, _, err := GetCircuit("")
		So(err, ShouldBeNil)
		return cb.executorPool.ActiveCount() == 0
	}

	Convey("with a run command that doesn't return", t, func() {
		Convey("checking many times that after GoC(context.Background(), ), the ticket returns to the pool after the timeout", func() {
			err := quick.Check(compareTicket, nil)
			So(err, ShouldBeNil)
		})
	})
}

func TestReturnTicket(t *testing.T) {
	Convey("with a run command that doesn't return", t, func() {
		defer Flush()

		ConfigureCommand("", CommandConfig{Timeout: 10})
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		errChan := GoC(ctx, "", func(ctx context.Context) error {
			//there are multiple ways to block here, the following sequence of steps
			//will block:: c := make(chan struct{});  <-c; return nil // should block
			//however, this would leak the internal GoC.func goroutine
			//another non-leaking way to do this would be to simply: return ErrTimeout
			c := make(chan struct{})
			<-c // should block (hence we add an exception in go-leak)
			return nil
		}, nil)

		Convey("after GoC(context.Background(), ), the ticket returns to the pool after the timeout", func() {
			err := <-errChan
			So(err, ShouldResemble, ErrTimeout)

			cb, _, err := GetCircuit("")
			So(err, ShouldBeNil)
			So(cb.executorPool.ActiveCount(), ShouldEqual, 0)
		})
	})
}

func TestContextHandling(t *testing.T) {
	Convey("with a run command which times out", t, func() {
		defer Flush()

		ConfigureCommand("", CommandConfig{Timeout: 15})
		cb, _, err := GetCircuit("")
		if err != nil {
			t.Fatal(err)
		}

		out := make(chan int, 1)
		run := func(ctx context.Context) error {
			time.Sleep(20 * time.Millisecond)
			out <- 1
			return nil
		}

		fallback := func(ctx context.Context, e error) error {
			return nil
		}

		Convey("with a valid context", func() {
			errChan := GoC(context.Background(), "", run, nil)
			time.Sleep(25 * time.Millisecond)
			So((<-errChan).Error(), ShouldEqual, ErrTimeout.Error())
			So(cb.metrics.DefaultCollector().NumRequests().Sum(time.Now()), ShouldEqual, 1)
			So(cb.metrics.DefaultCollector().Failures().Sum(time.Now()), ShouldEqual, 0)
			So(cb.metrics.DefaultCollector().Timeouts().Sum(time.Now()), ShouldEqual, 1)
			So(cb.metrics.DefaultCollector().ContextCanceled().Sum(time.Now()), ShouldEqual, 0)
			So(cb.metrics.DefaultCollector().ContextDeadlineExceeded().Sum(time.Now()), ShouldEqual, 0)
		})

		Convey("with a valid context and a fallback", func() {
			errChan := GoC(context.Background(), "", run, fallback)
			time.Sleep(25 * time.Millisecond)
			So(len(errChan), ShouldEqual, 0)
			So(cb.metrics.DefaultCollector().NumRequests().Sum(time.Now()), ShouldEqual, 1)
			So(cb.metrics.DefaultCollector().Failures().Sum(time.Now()), ShouldEqual, 0)
			So(cb.metrics.DefaultCollector().Timeouts().Sum(time.Now()), ShouldEqual, 1)
			So(cb.metrics.DefaultCollector().ContextCanceled().Sum(time.Now()), ShouldEqual, 0)
			So(cb.metrics.DefaultCollector().ContextDeadlineExceeded().Sum(time.Now()), ShouldEqual, 0)
			So(cb.metrics.DefaultCollector().FallbackSuccesses().Sum(time.Now()), ShouldEqual, 1)
		})

		Convey("with a context timeout", func() {
			testCtx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
			errChan := GoC(testCtx, "", run, nil)
			time.Sleep(25 * time.Millisecond)
			So((<-errChan).Error(), ShouldEqual, context.DeadlineExceeded.Error())
			So(cb.metrics.DefaultCollector().NumRequests().Sum(time.Now()), ShouldEqual, 1)
			So(cb.metrics.DefaultCollector().Failures().Sum(time.Now()), ShouldEqual, 0)
			So(cb.metrics.DefaultCollector().Timeouts().Sum(time.Now()), ShouldEqual, 0)
			So(cb.metrics.DefaultCollector().ContextCanceled().Sum(time.Now()), ShouldEqual, 0)
			So(cb.metrics.DefaultCollector().ContextDeadlineExceeded().Sum(time.Now()), ShouldEqual, 1)
			cancel()
		})

		Convey("with a context timeout and a fallback", func() {
			testCtx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
			errChan := GoC(testCtx, "", run, fallback)
			time.Sleep(25 * time.Millisecond)
			So(len(errChan), ShouldEqual, 0)
			So(cb.metrics.DefaultCollector().NumRequests().Sum(time.Now()), ShouldEqual, 1)
			So(cb.metrics.DefaultCollector().Failures().Sum(time.Now()), ShouldEqual, 0)
			So(cb.metrics.DefaultCollector().Timeouts().Sum(time.Now()), ShouldEqual, 0)
			So(cb.metrics.DefaultCollector().ContextCanceled().Sum(time.Now()), ShouldEqual, 0)
			So(cb.metrics.DefaultCollector().ContextDeadlineExceeded().Sum(time.Now()), ShouldEqual, 1)
			So(cb.metrics.DefaultCollector().FallbackSuccesses().Sum(time.Now()), ShouldEqual, 1)
			cancel()
		})

		Convey("with a canceled context", func() {
			testCtx, cancel := context.WithCancel(context.Background())
			errChan := GoC(testCtx, "", run, nil)
			time.Sleep(5 * time.Millisecond)
			cancel()
			time.Sleep(20 * time.Millisecond)
			So((<-errChan).Error(), ShouldEqual, context.Canceled.Error())
			So(cb.metrics.DefaultCollector().NumRequests().Sum(time.Now()), ShouldEqual, 1)
			So(cb.metrics.DefaultCollector().Failures().Sum(time.Now()), ShouldEqual, 0)
			So(cb.metrics.DefaultCollector().Timeouts().Sum(time.Now()), ShouldEqual, 0)
			So(cb.metrics.DefaultCollector().ContextCanceled().Sum(time.Now()), ShouldEqual, 1)
			So(cb.metrics.DefaultCollector().ContextDeadlineExceeded().Sum(time.Now()), ShouldEqual, 0)
		})

		Convey("with a canceled context and a fallback", func() {
			testCtx, cancel := context.WithCancel(context.Background())
			errChan := GoC(testCtx, "", run, fallback)
			time.Sleep(5 * time.Millisecond)
			cancel()
			time.Sleep(20 * time.Millisecond)
			So(len(errChan), ShouldEqual, 0)
			So(cb.metrics.DefaultCollector().NumRequests().Sum(time.Now()), ShouldEqual, 1)
			So(cb.metrics.DefaultCollector().Failures().Sum(time.Now()), ShouldEqual, 0)
			So(cb.metrics.DefaultCollector().Timeouts().Sum(time.Now()), ShouldEqual, 0)
			So(cb.metrics.DefaultCollector().ContextCanceled().Sum(time.Now()), ShouldEqual, 1)
			So(cb.metrics.DefaultCollector().ContextDeadlineExceeded().Sum(time.Now()), ShouldEqual, 0)
			So(cb.metrics.DefaultCollector().FallbackSuccesses().Sum(time.Now()), ShouldEqual, 1)
		})

	})
}

func TestDoC(t *testing.T) {
	Convey("with a command which succeeds", t, func() {
		defer Flush()

		out := make(chan bool, 1)
		run := func(ctx context.Context) error {
			out <- true
			return nil
		}

		Convey("the run function is executed", func() {
			err := DoC(context.Background(), "", run, nil)
			So(err, ShouldBeNil)
			So(<-out, ShouldEqual, true)
		})
	})

	Convey("with a command which fails", t, func() {
		defer Flush()

		run := func(ctx context.Context) error {
			return fmt.Errorf("i failed")
		}

		Convey("with no fallback", func() {
			err := DoC(context.Background(), "", run, nil)
			Convey("the error is returned", func() {
				So(err.Error(), ShouldEqual, "i failed")
			})
		})

		Convey("with a succeeding fallback", func() {
			out := make(chan bool, 1)
			fallback := func(ctx context.Context, err error) error {
				out <- true
				return nil
			}

			err := DoC(context.Background(), "", run, fallback)

			Convey("the fallback is executed", func() {
				So(err, ShouldBeNil)
				So(<-out, ShouldEqual, true)
			})
		})

		Convey("with a failing fallback", func() {
			fallback := func(ctx context.Context, err error) error {
				return fmt.Errorf("fallback failed")
			}

			err := DoC(context.Background(), "", run, fallback)

			Convey("both errors are returned", func() {
				So(err.Error(), ShouldEqual, "fallback failed with 'fallback failed'. run error was 'i failed'")
			})
		})
	})

	Convey("with a command which times out", t, func() {
		defer Flush()

		ConfigureCommand("", CommandConfig{Timeout: 10})

		err := DoC(context.Background(), "", func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		}, nil)

		Convey("the timeout error is returned", func() {
			So(err.Error(), ShouldEqual, "hystrix: timeout")
		})
	})
}
