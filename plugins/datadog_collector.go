package plugins

import (

	// Developed on https://github.com/DataDog/datadog-go/tree/a27810dd518c69be741a7fd5d0e39f674f615be8
	"github.com/DataDog/datadog-go/statsd"
	"github.com/afex/hystrix-go/hystrix/metric_collector"
)

// These metrics are constants because we're leveraging the Datadog tagging
// extension to statsd.
//
// They only apply to the DatadogCollector and are only useful if providing your
// own implemenation of DatadogClient
const (
	// DM = Datadog Metric
	DM_CircuitOpen       = "hystrix.circuitOpen"
	DM_Attempts          = "hystrix.attempts"
	DM_Errors            = "hystrix.errors"
	DM_Successes         = "hystrix.successes"
	DM_Failures          = "hystrix.failures"
	DM_Rejects           = "hystrix.rejects"
	DM_ShortCircuits     = "hystrix.shortCircuits"
	DM_Timeouts          = "hystrix.timeouts"
	DM_FallbackSuccesses = "hystrix.fallbackSuccesses"
	DM_FallbackFailures  = "hystrix.fallbackFailures"
	DM_TotalDuration     = "hystrix.totalDuration"
	DM_RunDuration       = "hystrix.runDuration"
)

type (
	// DatadogClient is the minimum interface needed by
	// NewDatadogCollectorWithClient
	DatadogClient interface {
		Count(name string, value int64, tags []string, rate float64) error
		Gauge(name string, value float64, tags []string, rate float64) error
		TimeInMilliseconds(name string, value float64, tags []string, rate float64) error
	}

	// DatadogCollector fulfills the metricCollector interface allowing users to
	// ship circuit stats to Datadog.
	//
	// This Collector, by default, uses github.com/DataDog/datadog-go/statsd for
	// transport. The main advantage of this over statsd is building graphs and
	// multi-alert monitors around single metrics (constantized above) and
	// adding tag dimensions. You can set up a single monitor to rule them all
	// across services and geographies. Graphs become much simpler to setup by
	// allowing you to create queries like the following
	//
	//   {
	//     "viz": "timeseries",
	//     "requests": [
	//       {
	//         "q": "max:hystrix.runDuration.95percentile{$region} by {hystrixcircuit}",
	//         "type": "line"
	//       }
	//     ]
	//   }
	//
	// As new circuits come online you get graphing and monitoring "for free".
	DatadogCollector struct {
		client DatadogClient
		tags   []string
	}
)

// NewDatadogCollector creates a collector for a specific circuit with a
// "github.com/DataDog/datadog-go/statsd".(*Client).
//
// addr is in the format "<host>:<port>" (e.g. "localhost:8125")
//
// prefix may be an empty string
//
// Example use
//  package main
//
//  import (
//  	"github.com/afex/hystrix-go/plugins"
//  	"github.com/afex/hystrix-go/hystrix/metric_collector"
//  )
//
//  func main() {
//  	collector, err := plugins.NewDatadogCollector("localhost:8125", "")
//  	if err != nil {
//  		panic(err)
//  	}
//  	metricCollector.Registry.Register(collector)
//  }
func NewDatadogCollector(addr, prefix string) (func(string) metricCollector.MetricCollector, error) {

	c, err := statsd.NewBuffered(addr, 100)
	if err != nil {
		return nil, err
	}

	// Prefix every metric with the app name
	c.Namespace = prefix

	return NewDatadogCollectorWithClient(c), nil
}

// NewDatadogCollectorWithClient accepts an interface which allows you to
// provide your own implementation of a statsd client, alter configuration on
// "github.com/DataDog/datadog-go/statsd".(*Client), provide additional tags per
// circuit-metric tuple, and add logging if you need it.
func NewDatadogCollectorWithClient(client DatadogClient) func(string) metricCollector.MetricCollector {

	return func(name string) metricCollector.MetricCollector {

		return &DatadogCollector{
			client: client,
			tags:   []string{"hystrixcircuit:" + name},
		}
	}
}

func (dc *DatadogCollector) Update(r metricCollector.MetricResult) {
	if r.Attempts > 0 {
		dc.client.Count(DM_Attempts, int64(r.Attempts), dc.tags, 1.0)
	}
	if r.Errors > 0 {
		dc.client.Count(DM_Errors, int64(r.Errors), dc.tags, 1.0)
	}
	if r.Successes > 0 {
		dc.client.Gauge(DM_CircuitOpen, 0, dc.tags, 1.0)
		dc.client.Count(DM_Successes, int64(r.Successes), dc.tags, 1.0)
	}
	if r.Failures > 0 {
		dc.client.Count(DM_Failures, int64(r.Failures), dc.tags, 1.0)
	}
	if r.Rejects > 0 {
		dc.client.Count(DM_Rejects, int64(r.Rejects), dc.tags, 1.0)
	}
	if r.ShortCircuits > 0 {
		dc.client.Gauge(DM_CircuitOpen, 1, dc.tags, 1.0)
		dc.client.Count(DM_ShortCircuits, int64(r.ShortCircuits), dc.tags, 1.0)
	}
	if r.Timeouts > 0 {
		dc.client.Count(DM_Timeouts, int64(r.Timeouts), dc.tags, 1.0)
	}
	if r.FallbackSuccesses > 0 {
		dc.client.Count(DM_FallbackSuccesses, int64(r.FallbackSuccesses), dc.tags, 1.0)
	}
	if r.FallbackFailures > 0 {
		dc.client.Count(DM_FallbackFailures, int64(r.FallbackFailures), dc.tags, 1.0)
	}

	ms := float64(r.TotalDuration.Nanoseconds() / 1000000)
	dc.client.TimeInMilliseconds(DM_TotalDuration, ms, dc.tags, 1.0)

	ms = float64(r.RunDuration.Nanoseconds() / 1000000)
	dc.client.TimeInMilliseconds(DM_RunDuration, ms, dc.tags, 1.0)
}

// Reset is a noop operation in this collector.
func (dc *DatadogCollector) Reset() {}
