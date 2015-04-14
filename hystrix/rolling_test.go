package hystrix

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMax(t *testing.T) {
	defer Flush()

	Convey("when adding values to a rolling number", t, func() {
		n := newRollingNumber()
		for _, x := range []int{10, 11, 9} {
			n.UpdateMax(x)
			time.Sleep(1 * time.Second)
		}

		Convey("it should know the maximum", func() {
			So(n.Max(time.Now()), ShouldEqual, 11)
		})
	})
}

func BenchmarkRollingNumberIncrement(b *testing.B) {
	n := newRollingNumber()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		n.Increment()
	}
}

func BenchmarkRollingNumberUpdateMax(b *testing.B) {
	n := newRollingNumber()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		n.UpdateMax(i)
	}
}
