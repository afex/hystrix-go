package hystrix

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConfigureConcurrency(t *testing.T) {
	Convey("given a command configured for 100 concurrent requests", t, func() {
		ConfigureCommand("", CommandConfig{MaxConcurrentRequests: 100})

		Convey("reading the concurrency should be the same", func() {
			So(getSettings("").MaxConcurrentRequests, ShouldEqual, 100)
		})
	})
}

func TestConfigureTimeout(t *testing.T) {
	Convey("given a command configured for a 10000 milliseconds", t, func() {
		ConfigureCommand("", CommandConfig{Timeout: 10000})

		Convey("reading the timeout should be the same", func() {
			So(getSettings("").Timeout, ShouldEqual, time.Duration(10*time.Second))
		})
	})
}

func TestConfigureRVT(t *testing.T) {
	Convey("given a command configured to need 30 requests before tripping the circuit", t, func() {
		ConfigureCommand("", CommandConfig{RequestVolumeThreshold: 30})

		Convey("reading the threshold should be the same", func() {
			So(getSettings("").RequestVolumeThreshold, ShouldEqual, uint64(30))
		})
	})
}

func TestSleepWindowDefault(t *testing.T) {
	Convey("given default settings", t, func() {
		ConfigureCommand("", CommandConfig{})

		Convey("the sleep window should be 5 seconds", func() {
			So(getSettings("").SleepWindow, ShouldEqual, time.Duration(5*time.Second))
		})
	})
}

func TestCircuitBreakerDisabledDefault(t *testing.T) {
	Convey("given default settings", t, func() {
		ConfigureCommand("", CommandConfig{})

		Convey("the circuit breaker disabled should be false", func() {
			So(getSettings("").CircuitBreakerDisabled, ShouldEqual, false)
		})
	})
}

func TestConfigCircuitBreakerDisabled(t *testing.T) {
	Convey("given a command configured to circuit breaker disabled true", t, func() {
		ConfigureCommand("", CommandConfig{CircuitBreakerDisabled: true})

		Convey("the circuit breaker disabled should be true", func() {
			So(getSettings("").CircuitBreakerDisabled, ShouldEqual, true)
		})
	})
}
