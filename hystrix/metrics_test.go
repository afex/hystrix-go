package hystrix

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func metricFailingPercent(p int, settings *SettingsCollection) *metricExchange {
	m := newMetricExchange("", settings)
	for i := 0; i < 100; i++ {
		t := "success"
		if i < p {
			t = "failure"
		}
		m.Updates <- &commandExecution{Types: []string{t}}
	}

	// Updates needs to be flushed
	time.Sleep(100 * time.Millisecond)

	return m
}

func TestErrorPercent(t *testing.T) {
	Convey("with a metric failing 40 percent of the time", t, func() {

		failingPercent := 40
		now := time.Now()

		Convey("ErrorPercent() should return 40", func() {
			m := metricFailingPercent(failingPercent, NewSettingsCollection())
			p := m.ErrorPercent(now)
			So(p, ShouldEqual, failingPercent)
		})

		Convey("and a error threshold set to 39", func() {

			settings := NewSettingsCollection()
			settings.ConfigureCommand("", CommandConfig{ErrorPercentThreshold: failingPercent - 1})

			m := metricFailingPercent(failingPercent, settings)

			Convey("the metrics should be unhealthy", func() {
				So(m.IsHealthy(now), ShouldBeFalse)
			})
		})
	})
}
