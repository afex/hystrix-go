package hystrix

import (
	"testing"
	"time"
)

func TestErrorPercent(t *testing.T) {
	m := NewMetrics()
	m.Updates <- &ExecutionMetric{Type: "success"}
	m.Updates <- &ExecutionMetric{Type: "success"}
	m.Updates <- &ExecutionMetric{Type: "success"}
	m.Updates <- &ExecutionMetric{Type: "failure"}
	m.Updates <- &ExecutionMetric{Type: "failure"}

	p := m.ErrorPercent(time.Now())
	if p != 0.40 {
		t.Fatalf("expected 0.40 percent errors, but got %v", p)
	}
}
