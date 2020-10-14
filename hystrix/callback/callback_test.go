package callback

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRegister(t *testing.T) {
	Convey("Register a command", t, func() {
		Register("Test-Command", func(name string, state State) {})

		Convey("Read Callback function for Registered Command", func() {
			callbackFunc, _ := circuitCallback["Test-Command"]
			So(callbackFunc, ShouldNotBeNil)
		})
		Convey("Read Callback function for unknown Command", func() {
			callbackFunc, _ := circuitCallback["Command"]
			So(callbackFunc, ShouldBeNil)
		})
	})
}

func TestInvoke(t *testing.T) {
	Convey("Register a command", t, func() {
		var callbackInvoked bool
		var callbackState State = Close
		Register("TestInvokeCommand", func(name string, state State) {
			callbackInvoked = true
			callbackState = state
		})

		Invoke("TestInvokeCommand", Open)
		time.Sleep(2 * time.Second)
		Convey("Invoke Callback for Registered Command", func() {
			So(callbackInvoked, ShouldBeTrue)
			So(callbackState, ShouldEqual, Open)
		})
	})

	Convey("Register a Invoke command", t, func() {
		var callbackInvoked = false
		var callbackState State = Close
		Register("TestInvokeCommand", func(name string, state State) {
			callbackInvoked = true
			callbackState = state
		})

		Invoke("Command", Open)

		Convey("Read Callback function for unknown Command", func() {
			So(callbackInvoked, ShouldBeFalse)
			So(callbackState, ShouldEqual, Close)
		})
	})
}
