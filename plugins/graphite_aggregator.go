// Plugins allows users to operate on statistics recorded for each circuit operation.
// Plugins should be careful to be lightweight as they will be called frequently.
package plugins

import (
	"net"
	"strings"
	"time"

	"github.com/afex/hystrix-go/hystrix/metric_collector"
	"github.com/rcrowley/go-metrics"
)

var makeTimerFunc = func() interface{} { return metrics.NewTimer() }
var makeCounterFunc = func() interface{} { return metrics.NewCounter() }

// GraphiteCollector fulfills the metricCollector interface allowing users to ship circuit
// stats to a graphite backend. To use users must call InitializeGraphiteCollector before
// circuits are started. Then register NewGraphiteCollector with metricCollector.Registry.Register(NewGraphiteCollector).
//
// This Collector uses github.com/rcrowley/go-metrics for aggregation. See that repo for more details
// on how metrics are aggregated and expressed in graphite.
type GraphiteCollector struct {
	attemptsPrefix          string
	errorsPrefix            string
	successesPrefix         string
	failuresPrefix          string
	rejectsPrefix           string
	shortCircuitsPrefix     string
	timeoutsPrefix          string
	fallbackSuccessesPrefix string
	fallbackFailuresPrefix  string
	totalDurationPrefix     string
	runDurationPrefix       string
}

// GraphiteCollectorConfig provides configuration that the graphite client will need.
type GraphiteCollectorConfig struct {
	// GraphiteAddr is the tcp address of the graphite server
	GraphiteAddr *net.TCPAddr
	// Prefix is the prefix that will be prepended to all metrics sent from this collector.
	Prefix string
	// TickInterval spcifies the period that this collector will send metrics to the server.
	TickInterval time.Duration
}

// InitializeGraphiteCollector creates the connection to the graphite server
// and should be called before any metrics are recorded.
func InitializeGraphiteCollector(config *GraphiteCollectorConfig) {
	go metrics.Graphite(metrics.DefaultRegistry, config.TickInterval, config.Prefix, config.GraphiteAddr)
}

// NewGraphiteCollector creates a collector for a specific circuit. The
// prefix given to this circuit will be {config.Prefix}.{circuit_name}.{metric}.
// Circuits with "/" in their names will have them replaced with ".".
func NewGraphiteCollector(name string) metricCollector.MetricCollector {
	name = strings.Replace(name, "/", "-", -1)
	name = strings.Replace(name, ":", "-", -1)
	name = strings.Replace(name, ".", "-", -1)
	return &GraphiteCollector{
		attemptsPrefix:          name + ".attempts",
		errorsPrefix:            name + ".errors",
		successesPrefix:         name + ".successes",
		failuresPrefix:          name + ".failures",
		rejectsPrefix:           name + ".rejects",
		shortCircuitsPrefix:     name + ".shortCircuits",
		timeoutsPrefix:          name + ".timeouts",
		fallbackSuccessesPrefix: name + ".fallbackSuccesses",
		fallbackFailuresPrefix:  name + ".fallbackFailures",
		totalDurationPrefix:     name + ".totalDuration",
		runDurationPrefix:       name + ".runDuration",
	}
}

func (g *GraphiteCollector) incrementCounterMetric(prefix string, i float64) {
	if i == 0 {
		return
	}
	c, ok := metrics.GetOrRegister(prefix, makeCounterFunc).(metrics.Counter)
	if !ok {
		return
	}
	c.Inc(int64(i))
}

func (g *GraphiteCollector) updateTimerMetric(prefix string, dur time.Duration) {
	c, ok := metrics.GetOrRegister(prefix, makeTimerFunc).(metrics.Timer)
	if !ok {
		return
	}
	c.Update(dur)
}

func (g *GraphiteCollector) Update(r metricCollector.MetricResult) {
	g.incrementCounterMetric(g.attemptsPrefix, r.Attempts)
	g.incrementCounterMetric(g.errorsPrefix, r.Errors)
	g.incrementCounterMetric(g.successesPrefix, r.Successes)
	g.incrementCounterMetric(g.failuresPrefix, r.Failures)
	g.incrementCounterMetric(g.rejectsPrefix, r.Rejects)
	g.incrementCounterMetric(g.shortCircuitsPrefix, r.ShortCircuits)
	g.incrementCounterMetric(g.timeoutsPrefix, r.Timeouts)
	g.incrementCounterMetric(g.fallbackSuccessesPrefix, r.FallbackSuccesses)
	g.incrementCounterMetric(g.fallbackFailuresPrefix, r.FallbackFailures)
	g.updateTimerMetric(g.totalDurationPrefix, r.TotalDuration)
	g.updateTimerMetric(g.runDurationPrefix, r.RunDuration)
}

// Reset is a noop operation in this collector.
func (g *GraphiteCollector) Reset() {}
