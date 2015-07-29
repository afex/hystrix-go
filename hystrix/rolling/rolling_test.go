package rolling

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMax(t *testing.T) {

	Convey("when adding values to a rolling number", t, func() {
		n := NewNumber()
		for _, x := range []float64{10, 11, 9} {
			n.UpdateMax(x)
			time.Sleep(1 * time.Second)
		}

		Convey("it should know the maximum", func() {
			So(n.Max(time.Now()), ShouldEqual, 11)
		})
	})
}

func TestAvg(t *testing.T) {
	Convey("when adding values to a rolling number", t, func() {
		n := NewNumber()
		for _, x := range []float64{0.5, 1.5, 2.5, 3.5, 4.5} {
			n.Increment(x)
			time.Sleep(1 * time.Second)
		}

		Convey("it should calculate the average over the number of configured buckets", func() {
			So(n.Avg(time.Now()), ShouldEqual, 1.25)
		})
	})
}

func BenchmarkRollingNumberIncrement(b *testing.B) {
	n := NewNumber()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		n.Increment(1)
	}
}

func BenchmarkRollingNumberUpdateMax(b *testing.B) {
	n := NewNumber()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		n.UpdateMax(float64(i))
	}
}
