package plugins

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSampleRate(t *testing.T) {
	Convey("when initializing the collector", t, func() {
		Convey("with no sample rate", func() {
			client, err := InitializeStatsdCollector(&StatsdCollectorConfig{
				StatsdAddr: "localhost:8125",
				Prefix:     "test",
			})
			So(err, ShouldBeNil)

			collector := client.NewStatsdCollector("foo").(*StatsdCollector)
			Convey("it defaults to no sampling", func() {
				So(collector.sampleRate, ShouldEqual, 1.0)
			})
		})
		Convey("with a sample rate", func() {
			client, err := InitializeStatsdCollector(&StatsdCollectorConfig{
				StatsdAddr: "localhost:8125",
				Prefix:     "test",
				SampleRate: 0.5,
			})
			So(err, ShouldBeNil)

			collector := client.NewStatsdCollector("foo").(*StatsdCollector)
			Convey("the rate is set", func() {
				So(collector.sampleRate, ShouldEqual, 0.5)
			})
		})
	})
}
