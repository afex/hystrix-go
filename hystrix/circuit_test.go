package hystrix

import (
	"testing"
	"time"
)

func TestOpen(t *testing.T) {
	c := NewCircuitBreaker("foo")

	for i := 0; i < 10; i++ {
		c.Health.Updates <- false
	}

	c.toggleOpenFromHealth(time.Now())

	if !c.IsOpen() {
		t.Fail()
	}
}

func TestClose(t *testing.T) {
	c := NewCircuitBreaker("foo")
	c.Open = true

	for i := 0; i < 10; i++ {
		c.Health.Updates <- true
	}

	c.toggleOpenFromHealth(time.Now())

	if c.IsOpen() {
		t.Fail()
	}
}
