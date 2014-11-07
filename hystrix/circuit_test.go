package hystrix

import "testing"
import "time"

func TestOpenBlah(t *testing.T) {
	c := NewCircuitBreaker()

	for i := 0; i < 10; i++ {
		c.health.Updates <- healthUpdate{false, time.Now()}
	}

	if !c.IsOpen() {
		t.Fail()
	}
}

// TODO: circuit re-closes when failures stop
func TestClose(t *testing.T) {

}
