package hystrix_go

import (
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	hystrixmetric "github.com/afex/hystrix-go/hystrix/metric_collector"
)

func SetupHystrixDashboard(service string, subType string) {
	namespace := fmt.Sprintf("%s_hystrix_%s", service, subType)
	registerHystrixPromMetricsCollector(namespace)
}

func registerHystrixPromMetricsCollector(namespace string) {
	wrapper := NewPrometheusCollector(namespace, nil, DefaultBuckets)
	// register and initialize to hystrix prometheus
	hystrixmetric.Registry.Register(wrapper.Collector)
}

func setupHystrixConfig(config HystrixConfig) {
	hystrix.ConfigureCommand(config.CommandName, hystrix.CommandConfig{
		Timeout:                config.Timeout,
		MaxConcurrentRequests:  config.MaxConcurrentRequests,
		RequestVolumeThreshold: config.RequestVolumeThreshold,
		SleepWindow:            config.SleepWindow,
		ErrorPercentThreshold:  config.ErrorPercentThreshold,
	})
}

func InitHystrixCommandConfigs(configs []HystrixConfig) {
	for _, config := range configs {
		setupHystrixConfig(config)
	}
}
