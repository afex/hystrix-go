package hystrix

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestReturn(t *testing.T) {
	defer Flush()

	Convey("when returning a ticket to the pool", t, func() {
		pool := newExecutorPool("pool")
		ticket := <-pool.Tickets
		pool.Return(ticket)
		time.Sleep(1 * time.Millisecond)
		Convey("total executed requests should increment", func() {
			So(pool.Metrics.Executed.Sum(time.Now()), ShouldEqual, 1)
		})
	})
}

func TestActiveCount(t *testing.T) {
	defer Flush()

	Convey("when 3 tickets are pulled", t, func() {
		pool := newExecutorPool("pool")
		<-pool.Tickets
		<-pool.Tickets
		ticket := <-pool.Tickets

		Convey("ActiveCount() should be 3", func() {
			So(pool.ActiveCount(), ShouldEqual, 3)
		})

		Convey("and one is returned", func() {
			pool.Return(ticket)

			Convey("max active requests should be 3", func() {
				time.Sleep(1 * time.Millisecond) // allow poolMetrics to process channel
				So(pool.Metrics.MaxActiveRequests.Max(time.Now()), ShouldEqual, 3)
			})
		})
	})
}
