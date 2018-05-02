package plugins

import (
	"log"
	"strings"
	"time"

	"github.com/afex/hystrix-go/hystrix/metric_collector"
	"github.com/cactus/go-statsd-client/statsd"
)

// StatsdCollector fulfills the metricCollector interface allowing users to ship circuit
// stats to a Statsd backend. To use users must call InitializeStatsdCollector before
// circuits are started. Then register NewStatsdCollector with metricCollector.Registry.Register(NewStatsdCollector).
//
// This Collector uses https://github.com/cactus/go-statsd-client/ for transport.
type StatsdCollector struct {
	client                  statsd.Statter
	circuitOpenPrefix       string
	attemptsPrefix          string
	errorsPrefix            string
	successesPrefix         string
	failuresPrefix          string
	rejectsPrefix           string
	shortCircuitsPrefix     string
	timeoutsPrefix          string
	fallbackSuccessesPrefix string
	fallbackFailuresPrefix  string
	canceledPrefix          string
	deadlinePrefix          string
	totalDurationPrefix     string
	runDurationPrefix       string
	concurrencyInUsePrefix  string
	sampleRate              float32
}

type StatsdCollectorClient struct {
	client     statsd.Statter
	sampleRate float32
}

// https://github.com/etsy/statsd/blob/master/docs/metric_types.md#multi-metric-packets
const (
	WANStatsdFlushBytes     = 512
	LANStatsdFlushBytes     = 1432
	GigabitStatsdFlushBytes = 8932
)

// StatsdCollectorConfig provides configuration that the Statsd client will need.
type StatsdCollectorConfig struct {
	// StatsdAddr is the tcp address of the Statsd server
	StatsdAddr string
	// Prefix is the prefix that will be prepended to all metrics sent from this collector.
	Prefix string
	// StatsdSampleRate sets statsd sampling. If 0, defaults to 1.0. (no sampling)
	SampleRate float32
	// FlushBytes sets message size for statsd packets. If 0, defaults to LANFlushSize.
	FlushBytes int
}

// InitializeStatsdCollector creates the connection to the Statsd server
// and should be called before any metrics are recorded.
//
// Users should ensure to call Close() on the client.
func InitializeStatsdCollector(config *StatsdCollectorConfig) (*StatsdCollectorClient, error) {
	flushBytes := config.FlushBytes
	if flushBytes == 0 {
		flushBytes = LANStatsdFlushBytes
	}

	sampleRate := config.SampleRate
	if sampleRate == 0 {
		sampleRate = 1
	}

	c, err := statsd.NewBufferedClient(config.StatsdAddr, config.Prefix, 1*time.Second, flushBytes)
	if err != nil {
		log.Printf("Could not initiale buffered client: %s. Falling back to a Noop Statsd client", err)
		c, _ = statsd.NewNoopClient()
	}
	return &StatsdCollectorClient{
		client:     c,
		sampleRate: sampleRate,
	}, err
}

// NewStatsdCollector creates a collector for a specific circuit. The
// prefix given to this circuit will be {config.Prefix}.{circuit_name}.{metric}.
// Circuits with "/" in their names will have them replaced with ".".
func (s *StatsdCollectorClient) NewStatsdCollector(name string) metricCollector.MetricCollector {
	if s.client == nil {
		log.Fatalf("Statsd client must be initialized before circuits are created.")
	}
	name = strings.Replace(name, "/", "-", -1)
	name = strings.Replace(name, ":", "-", -1)
	name = strings.Replace(name, ".", "-", -1)
	return &StatsdCollector{
		client:                  s.client,
		circuitOpenPrefix:       name + ".circuitOpen",
		attemptsPrefix:          name + ".attempts",
		errorsPrefix:            name + ".errors",
		successesPrefix:         name + ".successes",
		failuresPrefix:          name + ".failures",
		rejectsPrefix:           name + ".rejects",
		shortCircuitsPrefix:     name + ".shortCircuits",
		timeoutsPrefix:          name + ".timeouts",
		fallbackSuccessesPrefix: name + ".fallbackSuccesses",
		fallbackFailuresPrefix:  name + ".fallbackFailures",
		canceledPrefix:          name + ".contextCanceled",
		deadlinePrefix:          name + ".contextDeadlineExceeded",
		totalDurationPrefix:     name + ".totalDuration",
		runDurationPrefix:       name + ".runDuration",
		concurrencyInUsePrefix:  name + ".concurrencyInUse",
		sampleRate:              s.sampleRate,
	}
}

func (g *StatsdCollector) setGauge(prefix string, value int64) {
	err := g.client.Gauge(prefix, value, g.sampleRate)
	if err != nil {
		log.Printf("Error sending statsd metrics %s", prefix)
	}
}

func (g *StatsdCollector) incrementCounterMetric(prefix string, i float64) {
	if i == 0 {
		return
	}
	err := g.client.Inc(prefix, int64(i), g.sampleRate)
	if err != nil {
		log.Printf("Error sending statsd metrics %s", prefix)
	}
}

func (g *StatsdCollector) updateTimerMetric(prefix string, dur time.Duration) {
	err := g.client.TimingDuration(prefix, dur, g.sampleRate)
	if err != nil {
		log.Printf("Error sending statsd metrics %s", prefix)
	}
}

func (g *StatsdCollector) updateTimingMetric(prefix string, i int64) {
	err := g.client.Timing(prefix, i, g.sampleRate)
	if err != nil {
		log.Printf("Error sending statsd metrics %s", prefix)
	}
}

func (g *StatsdCollector) Update(r metricCollector.MetricResult) {
	if r.Successes > 0 {
		g.setGauge(g.circuitOpenPrefix, 0)
	} else if r.ShortCircuits > 0 {
		g.setGauge(g.circuitOpenPrefix, 1)
	}

	g.incrementCounterMetric(g.attemptsPrefix, r.Attempts)
	g.incrementCounterMetric(g.errorsPrefix, r.Errors)
	g.incrementCounterMetric(g.successesPrefix, r.Successes)
	g.incrementCounterMetric(g.failuresPrefix, r.Failures)
	g.incrementCounterMetric(g.rejectsPrefix, r.Rejects)
	g.incrementCounterMetric(g.shortCircuitsPrefix, r.ShortCircuits)
	g.incrementCounterMetric(g.timeoutsPrefix, r.Timeouts)
	g.incrementCounterMetric(g.fallbackSuccessesPrefix, r.FallbackSuccesses)
	g.incrementCounterMetric(g.fallbackFailuresPrefix, r.FallbackFailures)
	g.incrementCounterMetric(g.canceledPrefix, r.ContextCanceled)
	g.incrementCounterMetric(g.deadlinePrefix, r.ContextDeadlineExceeded)
	g.updateTimerMetric(g.totalDurationPrefix, r.TotalDuration)
	g.updateTimerMetric(g.runDurationPrefix, r.RunDuration)
	g.updateTimingMetric(g.concurrencyInUsePrefix, int64(100*r.ConcurrencyInUse))
}

// Reset is a noop operation in this collector.
func (g *StatsdCollector) Reset() {}
