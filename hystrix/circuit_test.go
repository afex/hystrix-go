package hystrix

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/songrgg/hystrix-go/hystrix/config"
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

func TestCircuitForceOpenAndClosed(t *testing.T) {
	defer Flush()

	Convey("when calling forceOpen", t, func() {
		var err error
		var cb *CircuitBreaker
		cb, _, err = GetCircuit("forceOpenTest")
		Convey("both circuit's forceOpen and forceClosed should be false after first initialization", func() {
			So(err, ShouldBeNil)
			So(config.GetSettings(cb.Name).ForceOpen, ShouldBeFalse)
			So(config.GetSettings(cb.Name).ForceClosed, ShouldBeFalse)
			So(cb.IsOpen(), ShouldBeFalse)
		})

		Convey("circuit's forceClosed should be false when forceOpen is true", func() {
			err := cb.ToggleForceOpen(true)
			So(err, ShouldBeNil)
			So(config.GetSettings(cb.Name).ForceOpen, ShouldBeTrue)
			So(config.GetSettings(cb.Name).ForceClosed, ShouldBeFalse)
			So(cb.IsOpen(), ShouldBeTrue)

			err = cb.ToggleForceOpen(false)
			So(err, ShouldBeNil)
			So(config.GetSettings(cb.Name).ForceOpen, ShouldBeFalse)
			So(config.GetSettings(cb.Name).ForceClosed, ShouldBeFalse)
			So(cb.IsOpen(), ShouldBeFalse)
		})

		Convey("circuit's forceOpen should be false when forceClosed is true", func() {
			err := cb.ToggleForceClosed(true)
			So(err, ShouldBeNil)
			So(config.GetSettings(cb.Name).ForceClosed, ShouldBeTrue)
			So(config.GetSettings(cb.Name).ForceOpen, ShouldBeFalse)
			So(cb.IsOpen(), ShouldBeFalse)

			err = cb.ToggleForceClosed(false)
			So(err, ShouldBeNil)
			So(config.GetSettings(cb.Name).ForceOpen, ShouldBeFalse)
			So(config.GetSettings(cb.Name).ForceClosed, ShouldBeFalse)
			So(cb.IsOpen(), ShouldBeFalse)
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

		config.ConfigureCommand("", config.CommandConfig{ErrorPercentThreshold: 50})

		cb, _, err := GetCircuit("")
		So(err, ShouldEqual, nil)
		So(cb.IsOpen(), ShouldBeFalse)
		openedTime := cb.openedOrLastTestedTime

		Convey("but the metrics are unhealthy", func() {
			cb.metrics = metricFailingPercent(100)
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
