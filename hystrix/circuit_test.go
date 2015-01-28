package hystrix

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetCircuit(t *testing.T) {
	defer Flush()

	Convey("when calling GetCircuit", t, func() {
		var created bool
		var err error
		_, created, err = GetCircuit("foo")

		Convey("once, the circuit should be created", func() {
			So(err, ShouldBeNil)
			So(created, ShouldEqual, true)
		})

		Convey("twice, the circuit should be reused", func() {
			_, created, err = GetCircuit("foo")
			So(err, ShouldBeNil)
			So(created, ShouldEqual, false)
		})
	})
}
