package hystrix

import (
	"testing"
	"time"
)

func metricFailingPercent(name string, p int) *Metrics {
	m := NewMetrics(name)
	for i := 0; i < 100; i++ {
		t := "success"
		if i < p {
			t = "failure"
		}
		m.Updates <- &ExecutionMetric{Type: t}
	}

	// Updates needs to be flushed
	time.Sleep(10 * time.Millisecond)

	return m
}

func TestErrorPercent(t *testing.T) {
	m := metricFailingPercent("error_percent", 40)

	p := m.ErrorPercent(time.Now())
	if p != 40 {
		t.Fatalf("expected 0.40 percent errors, but got %v", p)
	}
}

func TestIsHealthy(t *testing.T) {
	ConfigureCommand("healthy", CommandConfig{ErrorPercentThreshold: 39})

	m := metricFailingPercent("healthy", 40)
	now := time.Now()
	if m.IsHealthy(now) {
		t.Fatalf("expected metrics to be unhealthy, but wasn't at %v percent", m.ErrorPercent(now))
	}
}
