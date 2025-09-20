module github.com/afex/hystrix-go

go 1.15

require (
	github.com/DataDog/datadog-go v4.0.1+incompatible
	github.com/cactus/go-statsd-client/statsd v0.0.0-20200728222731-a2baea3bbfc6
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0
	github.com/smartystreets/goconvey v1.8.1
)

replace github.com/afex/hystrix-go => github.com/ContinuumLLC/hystrix-go v1.0.1
