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

func TestResizeTicketInUse(t *testing.T) {
	defer Flush()

	Convey("Given a pool of size 10 with one active ticket", t, func() {
		pool := newExecutorPool("pool")
		So(len(pool.Tickets), ShouldEqual, 10)
		<-pool.Tickets
		So(pool.ActiveCount(), ShouldEqual, 1)

		Convey("resizing it to 5 should result in", func() {
			pool.resize(5)
			Convey("An active count of 1", func() {
				So(pool.ActiveCount(), ShouldEqual, 1)
			})
			Convey("A pool size of 5", func() {
				So(pool.Max, ShouldEqual, 5)
			})
			Convey("And 4 available tickets", func() {
				So(len(pool.Tickets), ShouldEqual, 4)
			})
		})
	})
}

func TestResizeToSmall(t *testing.T) {
	defer Flush()

	Convey("Given a pool of size 10 with one active ticket", t, func() {
		pool := newExecutorPool("pool")
		So(len(pool.Tickets), ShouldEqual, 10)
		ticketa := <-pool.Tickets
		ticketb := <-pool.Tickets
		So(pool.ActiveCount(), ShouldEqual, 2)

		pool.resize(1)
		Convey("resizing it to 1 should result in", func() {
			Convey("An active count of 2", func() {
				So(pool.ActiveCount(), ShouldEqual, 2)
			})
			Convey("And a pool size of 1", func() {
				So(pool.Max, ShouldEqual, 1)
			})
		})

		Convey("Returning the active tickets should result in", func() {
			pool.Return(ticketa)
			pool.Return(ticketb)
			Convey("An active count of 0", func() {
				So(pool.ActiveCount(), ShouldEqual, 0)
			})
			Convey("And one available ticket", func() {
				So(len(pool.Tickets), ShouldEqual, 1)
			})
		})
	})
}
