package hystrix

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"math/rand"
	"testing/quick"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetCircuit(t *testing.T) {
	defer Flush()

	Convey("when calling GetCircuit", t, func() {
		var created bool
		var err error
		_, created, err = GetCircuit("foo")

		Convey("once, the circuit should be created", func() {
			So(err, ShouldBeNil)
			So(created, ShouldEqual, true)
		})

		Convey("twice, the circuit should be reused", func() {
			_, created, err = GetCircuit("foo")
			So(err, ShouldBeNil)
			So(created, ShouldEqual, false)
		})
	})
}

func TestMultithreadedGetCircuit(t *testing.T) {
	defer Flush()

	Convey("calling GetCircuit", t, func() {
		numThreads := 100
		var numCreates int32
		var numRunningRoutines int32
		var startingLine sync.WaitGroup
		var finishLine sync.WaitGroup
		startingLine.Add(1)
		finishLine.Add(numThreads)

		for i := 0; i < numThreads; i++ {
			go func() {
				if atomic.AddInt32(&numRunningRoutines, 1) == int32(numThreads) {
					startingLine.Done()
				} else {
					startingLine.Wait()
				}

				_, created, _ := GetCircuit("foo")

				if created {
					atomic.AddInt32(&numCreates, 1)
				}

				finishLine.Done()
			}()
		}

		finishLine.Wait()

		Convey("should be threadsafe", func() {
			So(numCreates, ShouldEqual, int32(1))
		})
	})
}

func TestReportEventOpenThenClose(t *testing.T) {
	Convey("when a circuit is closed", t, func() {
		defer Flush()
		ConfigureCommand("", CommandConfig{ErrorPercentThreshold: 50})

		cb, _, err := GetCircuit("")
		So(err, ShouldEqual, nil)
		So(cb.IsOpen(), ShouldBeFalse)
		openedTime := cb.openedOrLastTestedTime

		Convey("but the metrics are unhealthy", func() {
			cb.metrics = metricFailingPercentWithContext(cb.ctx, 100)
			So(cb.metrics.IsHealthy(time.Now()), ShouldBeFalse)

			Convey("and a success is reported", func() {
				err = cb.ReportEvent([]string{"success"}, time.Now(), 0)
				So(err, ShouldEqual, nil)

				Convey("the circuit does not open then close", func() {
					So(cb.openedOrLastTestedTime, ShouldEqual, openedTime)
				})
			})
		})
	})
}

func TestReportEventMultiThreaded(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	run := func() bool {
		defer Flush()
		// Make the circuit easily open and close intermittently.
		ConfigureCommand("", CommandConfig{
			MaxConcurrentRequests:  1,
			ErrorPercentThreshold:  1,
			RequestVolumeThreshold: 1,
			SleepWindow:            10,
		})
		cb, _, _ := GetCircuit("")
		count := 5
		wg := &sync.WaitGroup{}
		wg.Add(count)
		c := make(chan bool, count)
		for i := 0; i < count; i++ {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						t.Error(r)
						c <- false
					} else {
						wg.Done()
					}
				}()
				// randomized eventType to open/close circuit
				eventType := "rejected"
				if rand.Intn(3) == 1 {
					eventType = "success"
				}
				err := cb.ReportEvent([]string{eventType}, time.Now(), time.Second)
				if err != nil {
					t.Error(err)
				}
				time.Sleep(time.Millisecond)
				// cb.IsOpen() internally calls cb.setOpen()
				cb.IsOpen()
			}()
		}
		go func() {
			wg.Wait()
			c <- true
		}()
		return <-c
	}
	if err := quick.Check(run, nil); err != nil {
		t.Error(err)
	}
}
