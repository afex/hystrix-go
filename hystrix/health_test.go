package hystrix

import "testing"
import "time"

func Test10SecondWindow(t *testing.T) {
	health := NewHealth()

	for i := 0; i < 10; i++ {
		health.Updates <- false
	}

	now := time.Now()
	futureNow := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second()+11, 0, time.UTC)

	if !health.IsHealthy(futureNow) {
		t.Fail()
	}
}
