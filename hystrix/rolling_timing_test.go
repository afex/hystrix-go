package hystrix

import "testing"

func TestOrdinal(t *testing.T) {
	r := &RollingTiming{}
	if r.ordinal(5, 30) != 2 {
		t.Fatalf("expected ordinal of 2, got %v", r.ordinal(5, 30))
	}
	if r.ordinal(5, 40) != 2 {
		t.Fatalf("expected ordinal of 2, got %v", r.ordinal(5, 40))
	}
	if r.ordinal(5, 50) != 3 {
		t.Fatalf("expected ordinal of 3, got %v", r.ordinal(5, 50))
	}
}

func TestOrdinalLength1(t *testing.T) {
	r := &RollingTiming{}
	if r.ordinal(1, 0) != 1 {
		t.Fatalf("expected ordinal of 1, got %v", r.ordinal(1, 0))
	}	
}

func TestOrdinalLength2(t *testing.T) {
	r := &RollingTiming{}
	if r.ordinal(2, 50) != 1 {
		t.Fatalf("expected ordinal of 1, got %v", r.ordinal(2, 50))
	}
	if r.ordinal(2, 51) != 2 {
		t.Fatalf("expected ordinal of 1, got %v", r.ordinal(2, 50))
	}	
}