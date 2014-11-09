package hystrix

import "testing"

func TestOpenBlah(t *testing.T) {
	c := NewCircuitBreaker()

	for i := 0; i < 10; i++ {
		c.Health.Updates <- false
	}

	if !c.IsOpen() {
		t.Fail()
	}
}

// TODO: circuit re-closes when failures stop
func TestClose(t *testing.T) {

}
