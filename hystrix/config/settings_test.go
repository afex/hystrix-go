package config

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConfigureConcurrency(t *testing.T) {
	Convey("given a command configured for 100 concurrent requests", t, func() {
		ConfigureCommand("", CommandConfig{MaxConcurrentRequests: 100})

		Convey("reading the concurrency should be the same", func() {
			So(GetSettings("").MaxConcurrentRequests, ShouldEqual, 100)
		})
	})
}

func TestConfigureTimeout(t *testing.T) {
	Convey("given a command configured for a 10000 milliseconds", t, func() {
		ConfigureCommand("", CommandConfig{Timeout: 10000})

		Convey("reading the timeout should be the same", func() {
			So(GetSettings("").Timeout, ShouldEqual, time.Duration(10*time.Second))
		})
	})
}

func TestConfigureRVT(t *testing.T) {
	Convey("given a command configured to need 30 requests before tripping the circuit", t, func() {
		ConfigureCommand("", CommandConfig{RequestVolumeThreshold: 30})

		Convey("reading the threshold should be the same", func() {
			So(GetSettings("").RequestVolumeThreshold, ShouldEqual, uint64(30))
		})
	})
}

func TestSleepWindowDefault(t *testing.T) {
	Convey("given default settings", t, func() {
		ConfigureCommand("", CommandConfig{})

		Convey("the sleep window should be 5 seconds", func() {
			So(GetSettings("").SleepWindow, ShouldEqual, time.Duration(5*time.Second))
		})
	})
}

func TestConfigureCircuitBreakerEnabled(t *testing.T) {
	Convey("given a command configured for circuit breaker enabled", t, func() {
		ConfigureCommand("", CommandConfig{})

		Convey("the enabled should be the true by default", func() {
			So(GetSettings("").Enabled, ShouldBeTrue)
		})

		ConfigureCommand("", CommandConfig{Enabled: "false"})

		Convey("reading the enabled should be the false", func() {
			So(GetSettings("").Enabled, ShouldBeFalse)
		})

		ConfigureCommand("", CommandConfig{Enabled: "true"})

		Convey("reading the enabled should be the true", func() {
			So(GetSettings("").Enabled, ShouldBeTrue)
		})
	})
}

func TestConfigureMetrics(t *testing.T) {
	Convey("given a command configured for circuit breaker metrics rolling window to 20s and buckets to 20", t, func() {
		ConfigureCommand("", CommandConfig{MetricsRollingStatisticalWindow: 20000, MetricsRollingStatisticalWindowBuckets: 20})

		Convey("reading the metrics", func() {
			So(GetSettings("").MetricsRollingStatisticalWindow, ShouldEqual, time.Duration(20*time.Second))
			So(GetSettings("").MetricsRollingStatisticalWindowBuckets, ShouldEqual, 20)
		})
	})
}

func TestConfigureForceOpenAndForceClose(t *testing.T) {
	Convey("given a command configured for circuit breaker forceOpen and forceClosed", t, func() {
		ConfigureCommand("", CommandConfig{})

		Convey("default forceOpen and forceClosed should be false", func() {
			So(GetSettings("").ForceOpen, ShouldBeFalse)
			So(GetSettings("").ForceClosed, ShouldBeFalse)
		})

		ConfigureCommand("", CommandConfig{ForceOpen: true})

		Convey("default forceOpen should be true", func() {
			So(GetSettings("").ForceOpen, ShouldBeTrue)
			So(GetSettings("").ForceClosed, ShouldBeFalse)
		})

		ConfigureCommand("", CommandConfig{ForceClosed: true})

		Convey("default forceClosed should be true", func() {
			So(GetSettings("").ForceOpen, ShouldBeFalse)
			So(GetSettings("").ForceClosed, ShouldBeTrue)
		})

		Convey("configure should panic", func() {
			So(GetSettings("").ForceOpen, ShouldBeFalse)
			So(func() {
				ConfigureCommand("", CommandConfig{ForceClosed: true, ForceOpen: true})
			}, ShouldPanicWith, "forceOpen and forceClosed have conflicts")
		})
	})
}

func TestGetCircuitSettings(t *testing.T) {
	Convey("when calling GetCircuitSettings", t, func() {
		ConfigureCommand("test", CommandConfig{Timeout: 30000})

		Convey("should read the same setting just added", func() {
			So(GetCircuitSettings()["test"], ShouldEqual, GetSettings("test"))
			So(GetCircuitSettings()["test"].Timeout, ShouldEqual, time.Duration(30*time.Second))
		})
	})
}
