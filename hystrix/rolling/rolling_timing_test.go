package rolling

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestOrdinal(t *testing.T) {
	Convey("given a new rolling timing", t, func() {

		r := NewTiming()

		Convey("Mean() should be 0", func() {
			So(r.Mean(), ShouldEqual, 0)
		})

		Convey("and given a set of lengths and percentiles", func() {
			var ordinalTests = []struct {
				length   int
				perc     float64
				expected int64
			}{
				{1, 0, 1},
				{2, 0, 1},
				{2, 50, 1},
				{2, 51, 2},
				{5, 30, 2},
				{5, 40, 2},
				{5, 50, 3},
				{11, 25, 3},
				{11, 50, 6},
				{11, 75, 9},
				{11, 100, 11},
			}

			Convey("each should generate the expected ordinal", func() {

				for _, s := range ordinalTests {
					So(r.ordinal(s.length, s.perc), ShouldEqual, s.expected)
				}
			})
		})

		Convey("after adding 2 timings", func() {
			r.Add(100 * time.Millisecond)
			time.Sleep(2 * time.Second)
			r.Add(200 * time.Millisecond)

			Convey("the mean should be the average of the timings", func() {
				So(r.Mean(), ShouldEqual, 150)
			})
		})

		Convey("after adding many timings", func() {
			durations := []int{1, 1004, 1004, 1004, 1004, 1004, 1004, 1004, 1004, 1004, 1005, 1005, 1005, 1005, 1005, 1005, 1005, 1005, 1005, 1005, 1005, 1005, 1005, 1005, 1006, 1006, 1006, 1006, 1007, 1007, 1007, 1008, 1015}
			for _, d := range durations {
				r.Add(time.Duration(d) * time.Millisecond)
			}

			Convey("calculates correct percentiles", func() {
				So(r.Percentile(0), ShouldEqual, 1)
				So(r.Percentile(75), ShouldEqual, 1006)
				So(r.Percentile(99), ShouldEqual, 1015)
				So(r.Percentile(100), ShouldEqual, 1015)
			})
		})
	})
}
