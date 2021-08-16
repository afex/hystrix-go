package hystrix

import (
	"context"
	"testing"
	"time"

	"go.uber.org/goleak"

	. "github.com/smartystreets/goconvey/convey"
)

func metricFailingPercent(p int) *metricExchange {
	return metricFailingPercentWithContext(context.Background(), p)
}

func metricFailingPercentWithContext(ctx context.Context, p int) *metricExchange {
	m := newMetricExchange(ctx, "")
	for i := 0; i < 100; i++ {
		t := "success"
		if i < p {
			t = "failure"
		}
		m.Updates <- &commandExecution{Types: []string{t}}
	}

	// updates need to be flushed
	time.Sleep(100 * time.Millisecond)

	return m
}

func TestErrorPercent(t *testing.T) {
	Convey("with a metric failing 40 percent of the time", t, func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		m := metricFailingPercentWithContext(ctx, 40)
		now := time.Now()

		Convey("ErrorPercent() should return 40", func() {
			p := m.ErrorPercent(now)
			So(p, ShouldEqual, 40)
		})

		Convey("and a error threshold set to 39", func() {
			ConfigureCommand("", CommandConfig{ErrorPercentThreshold: 39})

			Convey("the metrics should be unhealthy", func() {
				So(m.IsHealthy(now), ShouldBeFalse)
			})

		})
	})
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("time.Sleep"),                                                             //tests that sleep in goroutines explicitly
		goleak.IgnoreTopFunction("github.com/afex/hystrix-go/hystrix.TestReturnTicket.func1.1"),            //explicit leak
		goleak.IgnoreTopFunction("github.com/afex/hystrix-go/hystrix.TestReturnTicket_QuickCheck.func1.1"), //explicit leak
	)
}
