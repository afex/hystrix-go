package hystrix

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetCircuit(t *testing.T) {
	defer FlushMetrics()

	Convey("when calling GetCircuit", t, func() {
		var created bool
		var err error
		_, created, err = GetCircuit("foo")
		if err != nil {
			t.Fatal(err)
		}

		Convey("once, the circuit should be created", func() {
			So(created, ShouldEqual, true)
		})

		Convey("twice, the circuit should be reused", func() {
			_, created, err = GetCircuit("foo")
			if err != nil {
				t.Fatal(err)
			}
			So(created, ShouldEqual, false)
		})
	})
}
