package hystrix

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestReturn(t *testing.T) {
	defer FlushMetrics()

	Convey("when returning a ticket to the pool", t, func() {
		pool := NewExecutorPool("pool")
		ticket := <-pool.Tickets
		pool.Return(ticket)
		Convey("total executed requests should increment", func() {
			So(pool.Metrics.Executed.Sum(time.Now()), ShouldEqual, 1)
		})
	})
}

func TestActiveCount(t *testing.T) {
	defer FlushMetrics()

	Convey("when 2 tickets are pulled", t, func() {
		pool := NewExecutorPool("pool")
		<-pool.Tickets
		<-pool.Tickets

		Convey("ActiveCount() should be 2", func() {
			So(pool.ActiveCount(), ShouldEqual, 2)
		})
	})
}
