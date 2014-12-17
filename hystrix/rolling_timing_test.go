package hystrix

import (
	"testing"
	"time"
)

func TestOrdinal(t *testing.T) {
	r := NewRollingTiming()
	var ordinalTests = []struct {
		length   int
		perc     float64
		expected int64
	}{
		{1, 0, 1},
		{2, 0, 1},
		{2, 50, 1},
		{2, 51, 2},
		{5, 30, 2},
		{5, 40, 2},
		{5, 50, 3},
		{11, 25, 3},
		{11, 50, 6},
		{11, 75, 9},
		{11, 100, 11},
	}
	for _, s := range ordinalTests {
		o := r.ordinal(s.length, s.perc)
		if o != s.expected {
			t.Fatalf("expected ordinal for length=%v, percentile=%v to be %v, got %v", s.length, s.perc, s.expected, o)
		}
	}
}

func TestNilMean(t *testing.T) {
	r := NewRollingTiming()
	if r.Mean() != 0 {
		t.Fatalf("expected nil mean to == 0")
	}
}

func TestMean(t *testing.T) {
	r := NewRollingTiming()

	r.Add(100 * time.Millisecond)
	time.Sleep(2 * time.Second)
	r.Add(200 * time.Millisecond)

	m := r.Mean()
	if m != 150 {
		t.Fatalf("expected 150 average, got %v", m)
	}
}
