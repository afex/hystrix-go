package hystrix_go

import (
	metricCollector "github.com/afex/hystrix-go/hystrix/metric_collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func NewTestRegistry() *prometheus.Registry {
	return prometheus.NewRegistry()
}

func TestPrometheusCollectorInitialization(t *testing.T) {
	reg := NewTestRegistry()
	pc := NewPrometheusCollector("test_namespace", reg, nil)

	assert.NotNil(t, pc)
	assert.NotNil(t, pc.circuitOpen)
	assert.NotNil(t, pc.attempts)
	assert.NotNil(t, pc.errors)
	assert.NotNil(t, pc.successes)
	assert.NotNil(t, pc.failures)
	assert.NotNil(t, pc.rejects)
	assert.NotNil(t, pc.shortCircuits)
	assert.NotNil(t, pc.timeouts)
	assert.NotNil(t, pc.fallbackSuccesses)
	assert.NotNil(t, pc.fallbackFailures)
	assert.NotNil(t, pc.contextCanceled)
	assert.NotNil(t, pc.contextDeadlineExceeded)
	assert.NotNil(t, pc.totalDuration)
	assert.NotNil(t, pc.runDuration)
	assert.NotNil(t, pc.concurrencyInUse)
}

func TestCollectorInitialization(t *testing.T) {
	pc := NewPrometheusCollector("test_namespace", nil, nil)
	collector := pc.Collector("test_command")

	assert.NotNil(t, collector)
	assert.IsType(t, &cmdCollector{}, collector)
}

func TestCollectorInitializesCounters(t *testing.T) {
	const commandName = "test_command"

	registry := prometheus.NewRegistry()
	pc := NewPrometheusCollector("test_namespace", registry, nil)
	collector := pc.Collector(commandName).(*cmdCollector)

	assert.Equal(t, float64(0), testutil.ToFloat64(collector.metrics.circuitOpen.WithLabelValues(commandName)))
	assert.Equal(t, float64(0), testutil.ToFloat64(collector.metrics.attempts.WithLabelValues(commandName)))
	assert.Equal(t, float64(0), testutil.ToFloat64(collector.metrics.errors.WithLabelValues(commandName)))

	registry.Unregister(collector.metrics.circuitOpen)
	registry.Unregister(collector.metrics.attempts)
	registry.Unregister(collector.metrics.errors)
}

func TestCollectorUpdatesMetrics(t *testing.T) {
	// Arrange
	const commandName = "test_command"
	registry := prometheus.NewRegistry()
	pc := NewPrometheusCollector("test_namespace", registry, nil)
	collector := pc.Collector(commandName).(*cmdCollector)

	result := metricCollector.MetricResult{
		Attempts:                1,
		Errors:                  1,
		Successes:               1,
		Failures:                1,
		Rejects:                 1,
		ShortCircuits:           1,
		Timeouts:                1,
		FallbackSuccesses:       1,
		FallbackFailures:        1,
		ContextCanceled:         1,
		ContextDeadlineExceeded: 1,
		TotalDuration:           time.Second,
		RunDuration:             time.Second,
	}

	// Act
	collector.Update(result)

	// Assert
	assert.Equal(t, float64(1), testutil.ToFloat64(collector.metrics.attempts.WithLabelValues(commandName)))
	assert.Equal(t, float64(1), testutil.ToFloat64(collector.metrics.errors.WithLabelValues(commandName)))
	assert.Equal(t, float64(1), testutil.ToFloat64(collector.metrics.successes.WithLabelValues(commandName)))
	assert.Equal(t, float64(1), testutil.ToFloat64(collector.metrics.failures.WithLabelValues(commandName)))
	assert.Equal(t, float64(1), testutil.ToFloat64(collector.metrics.rejects.WithLabelValues(commandName)))
	assert.Equal(t, float64(1), testutil.ToFloat64(collector.metrics.shortCircuits.WithLabelValues(commandName)))
	assert.Equal(t, float64(1), testutil.ToFloat64(collector.metrics.timeouts.WithLabelValues(commandName)))
	assert.Equal(t, float64(1), testutil.ToFloat64(collector.metrics.fallbackSuccesses.WithLabelValues(commandName)))
	assert.Equal(t, float64(1), testutil.ToFloat64(collector.metrics.fallbackFailures.WithLabelValues(commandName)))
	assert.Equal(t, float64(1), testutil.ToFloat64(collector.metrics.contextCanceled.WithLabelValues(commandName)))
	assert.Equal(t, float64(1), testutil.ToFloat64(collector.metrics.contextDeadlineExceeded.WithLabelValues(commandName)))

	// For histograms, verify using metric gathering
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	for _, metricFamily := range metricFamilies {
		if metricFamily.GetName() == "test_namespace_total_duration_seconds" {
			metric := metricFamily.GetMetric()[0]
			assert.Equal(t, float64(1), metric.GetHistogram().GetSampleSum())
		}
		if metricFamily.GetName() == "test_namespace_run_duration_seconds" {
			metric := metricFamily.GetMetric()[0]
			assert.Equal(t, float64(1), metric.GetHistogram().GetSampleSum())
		}
	}

	// Clean up
	metrics := []prometheus.Collector{
		collector.metrics.attempts,
		collector.metrics.errors,
		collector.metrics.successes,
		collector.metrics.failures,
		collector.metrics.rejects,
		collector.metrics.shortCircuits,
		collector.metrics.timeouts,
		collector.metrics.fallbackSuccesses,
		collector.metrics.fallbackFailures,
		collector.metrics.contextCanceled,
		collector.metrics.contextDeadlineExceeded,
		collector.metrics.totalDuration,
		collector.metrics.runDuration,
	}
	for _, m := range metrics {
		registry.Unregister(m)
	}
}
