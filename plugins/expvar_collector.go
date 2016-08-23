package plugins

import (
	"expvar"
	"github.com/afex/hystrix-go/hystrix/metric_collector"
	"strings"
	"time"
)

var (
	// We have a collector for every circuit breaker
	ExpVarCollectors map[string]*ExpvarCollector
)

func init() {
	ExpVarCollectors = make(map[string]*ExpvarCollector)
}

// ExpvarCollector fulfills the metricCollector interface allowing users to ship circuit
// stats to a Expvar backend. To use users must call InitializeExpvarCollector before
// circuits are started. Then register NewExpvarCollector with metricCollector.Registry.Register(NewExpvarCollector).
type ExpvarCollector struct {
	circuitOpen       *expvar.Int
	attempts          *expvar.Int
	errors            *expvar.Int
	successes         *expvar.Int
	failures          *expvar.Int
	rejects           *expvar.Int
	shortCircuits     *expvar.Int
	timeouts          *expvar.Int
	fallbackSuccesses *expvar.Int
	fallbackFailures  *expvar.Int
	totalDuration     *expvar.Float
	runDuration       *expvar.Float
}

type ExpvarCollectorClient struct {
	config *ExpvarCollectorConfig
}

// ExpvarCollectorConfig provides configuration that the Expvar client will need.
type ExpvarCollectorConfig struct {
	// Prefix is the prefix that will be prepended to all metrics sent from this collector.
	Prefix string
}

// InitializeExpvarCollector should be called before any metrics are recorded.
func InitializeExpvarCollector(config *ExpvarCollectorConfig) (*ExpvarCollectorClient, error) {
	return &ExpvarCollectorClient{config: config}, nil
}

// NewExpvarCollector creates a collector for a specific circuit. The
// prefix given to this circuit will be {config.Prefix}.{circuit_name}.{metric}.
// Circuits with "/" in their names will have them replaced with "-".
func (s *ExpvarCollectorClient) NewExpvarCollector(name string) metricCollector.MetricCollector {
	name = strings.Replace(name, "/", "-", -1)
	name = strings.Replace(name, ":", "-", -1)
	name = strings.Replace(name, ".", "-", -1)

	if s.config.Prefix != "" {
		name = s.config.Prefix + "." + name
	}

	collector, ok := ExpVarCollectors[name]
	if !ok {
		collector = &ExpvarCollector{
			circuitOpen:       expvar.NewInt(name + ".circuitOpen"),
			attempts:          expvar.NewInt(name + ".attempts"),
			errors:            expvar.NewInt(name + ".errors"),
			successes:         expvar.NewInt(name + ".successes"),
			failures:          expvar.NewInt(name + ".failures"),
			rejects:           expvar.NewInt(name + ".rejects"),
			shortCircuits:     expvar.NewInt(name + ".shortCircuits"),
			timeouts:          expvar.NewInt(name + ".timeouts"),
			fallbackSuccesses: expvar.NewInt(name + ".fallbackSuccesses"),
			fallbackFailures:  expvar.NewInt(name + ".fallbackFailures"),
			totalDuration:     expvar.NewFloat(name + ".totalDuration"),
			runDuration:       expvar.NewFloat(name + ".runDuration"),
		}

		ExpVarCollectors[name] = collector
	}

	return collector
}

// IncrementAttempts increments the number of calls to this circuit.
// This registers as a counter in the Expvar collector.
func (g *ExpvarCollector) IncrementAttempts() {
	g.attempts.Add(1)
}

// IncrementErrors increments the number of unsuccessful attempts.
// Attempts minus Errors will equal successes within a time range.
// Errors are any result from an attempt that is not a success.
// This registers as a counter in the Expvar collector.
func (g *ExpvarCollector) IncrementErrors() {
	g.errors.Add(1)
}

// IncrementSuccesses increments the number of requests that succeed.
// This registers as a counter in the Expvar collector.
func (g *ExpvarCollector) IncrementSuccesses() {
	g.circuitOpen.Set(0)
	g.successes.Add(1)
}

// IncrementFailures increments the number of requests that fail.
// This registers as a counter in the Expvar collector.
func (g *ExpvarCollector) IncrementFailures() {
	g.failures.Add(1)
}

// IncrementRejects increments the number of requests that are rejected.
// This registers as a counter in the Expvar collector.
func (g *ExpvarCollector) IncrementRejects() {
	g.rejects.Add(1)
}

// IncrementShortCircuits increments the number of requests that short circuited due to the circuit being open.
// This registers as a counter in the Expvar collector.
func (g *ExpvarCollector) IncrementShortCircuits() {
	g.circuitOpen.Set(1)
	g.shortCircuits.Add(1)
}

// IncrementTimeouts increments the number of timeouts that occurred in the circuit breaker.
// This registers as a counter in the Expvar collector.
func (g *ExpvarCollector) IncrementTimeouts() {
	g.timeouts.Add(1)
}

// IncrementFallbackSuccesses increments the number of successes that occurred during the execution of the fallback function.
// This registers as a counter in the Expvar collector.
func (g *ExpvarCollector) IncrementFallbackSuccesses() {
	g.fallbackSuccesses.Add(1)
}

// IncrementFallbackFailures increments the number of failures that occurred during the execution of the fallback function.
// This registers as a counter in the Expvar collector.
func (g *ExpvarCollector) IncrementFallbackFailures() {
	g.fallbackFailures.Add(1)
}

// UpdateTotalDuration updates the internal counter of how long we've run for.
// This registers as a timer in the Expvar collector.
func (g *ExpvarCollector) UpdateTotalDuration(timeSinceStart time.Duration) {
	g.totalDuration.Add(timeSinceStart.Seconds())
}

// UpdateRunDuration updates the internal counter of how long the last run took.
// This registers as a timer in the Expvar collector.
func (g *ExpvarCollector) UpdateRunDuration(runDuration time.Duration) {
	g.runDuration.Set(runDuration.Seconds())
}

// Reset is a noop operation in this collector.
func (g *ExpvarCollector) Reset() {}
