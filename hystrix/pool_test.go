package hystrix

import (
	"testing"
	"time"

	"sync/atomic"

	. "github.com/smartystreets/goconvey/convey"
)

func TestReturn(t *testing.T) {
	defer Flush()

	Convey("when returning a ticket to the pool", t, func() {
		pool := newBufferedExecutorPool("pool")
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
		pool := newBufferedExecutorPool("pool")
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

func TestWaitingCount(t *testing.T) {
	defer Flush()

	ConfigureCommand("pool", CommandConfig{QueueSizeRejectionThreshold: 50})
	Convey("when all execution tickets are pulled and then replenished", t, func() {

		pool := newBufferedExecutorPool("pool")
		checkpoint := make(chan struct{}, 1)
		completedTask := int32(0)
		// take away all pool tickets
		for i := 0; i < pool.Max; i++ {
			<-pool.Tickets
		}
		for i := 0; i < pool.QueueSizeRejectionThreshold-5; i++ {
			<-pool.WaitingTicket
			go func() {
				ticket := <-pool.Tickets

				if ticket != nil {
					taskN := atomic.AddInt32(&completedTask, 1)
					if taskN == 5 {
						close(checkpoint)
					}
					pool.ReturnWaitingTicket(&struct{}{})
				}
			}()
		}
		Convey("WaitingTicket should be max-5", func() {
			So(pool.WaitingCount(), ShouldEqual, pool.QueueSizeRejectionThreshold-5)
		})

		// return back 5 tickets
		for i := 0; i < 5; i++ {
			pool.Return(&struct{}{})
		}

		Convey("WaitingTicket should be max-10", func() {
			<-checkpoint
			So(pool.WaitingCount(), ShouldEqual, pool.QueueSizeRejectionThreshold-10)
		})

		for i := 0; i < 10; i++ {
			<-pool.WaitingTicket
		}
		Convey("WaitingTicket should be max (exhausted)", func() {
			So(pool.WaitingCount(), ShouldEqual, pool.QueueSizeRejectionThreshold)
		})
	})
}

func TestRefillTicket(t *testing.T) {
	defer Flush()

	ConfigureCommand("pool", CommandConfig{QueueSizeRejectionThreshold: 50})
	Convey("when all execution tickets are pulled and then replenished twice", t, func() {

		pool := newBufferedExecutorPool("pool")
		checkpoint1 := make(chan struct{}, 1)
		checkpoint2 := make(chan struct{}, 1)
		completedTask := int32(0)

		waitingTask := func() {
			<-pool.WaitingTicket
			go func() {
				ticket := <-pool.Tickets
				if ticket != nil {
					taskN := atomic.AddInt32(&completedTask, 1)
					if taskN == 5 {
						close(checkpoint1)
					} else if taskN == 10 {
						close(checkpoint2)
					}
					pool.ReturnWaitingTicket(&struct{}{})
				}
			}()
		}
		// take away all pool tickets
		for i := 0; i < pool.Max; i++ {
			<-pool.Tickets
		}
		for i := 0; i < pool.QueueSizeRejectionThreshold; i++ {
			waitingTask()
		}
		// complete 5 task
		for i := 0; i < 5; i++ {
			pool.Return(&struct{}{})
		}
		Convey("WaitingTicket should be max-5", func() {
			<-checkpoint1
			So(pool.ActiveCount(), ShouldEqual, pool.Max)
			So(pool.WaitingCount(), ShouldEqual, pool.QueueSizeRejectionThreshold-5)
		})

		// take away all pool tickets again
		for i := 0; i < 5; i++ {
			waitingTask()
		}
		// complete 5 task
		for i := 0; i < 5; i++ {
			pool.Return(&struct{}{})
		}
		Convey("WaitingTicket should be max-5 again", func() {
			<-checkpoint2
			So(pool.ActiveCount(), ShouldEqual, pool.Max)
			So(pool.WaitingCount(), ShouldEqual, pool.QueueSizeRejectionThreshold-5)
		})
	})
}
