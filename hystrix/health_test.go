package hystrix

import "testing"
import "time"

func Test10SecondWindow(t *testing.T) {
	health := NewHealth()

	for i := 0; i < 10; i++ {
		t := time.Now()

		health.Updates <- Update{false, time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second()-11, 0, time.UTC)}
	}

	if !health.IsHealthy() {
		t.Fail()
	}
}
