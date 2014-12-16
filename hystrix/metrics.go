package hystrix

import (
	"log"
	"time"
)

type ExecutionMetric struct {
	Type          string
	Time          time.Time
	TotalDuration time.Duration
	RunDuration   time.Duration
}

type Metrics struct {
	Updates chan *ExecutionMetric

	Count  *RollingNumber
	Errors *RollingNumber

	TotalDuration *RollingTiming
	RunDuration   *RollingTiming
}

func NewMetrics() *Metrics {
	m := &Metrics{}

	m.Updates = make(chan *ExecutionMetric)

	m.Count = NewRollingNumber()
	m.Errors = NewRollingNumber()
	m.TotalDuration = NewRollingTiming()
	m.RunDuration = NewRollingTiming()

	go m.Monitor()

	return m
}

func (m *Metrics) Monitor() {
	for update := range m.Updates {
		log.Printf("%+v", *update)

		m.Count.Increment()
		if update.Type != "success" {
			m.Errors.Increment()
		}

		m.TotalDuration.Add(update.TotalDuration)
		m.RunDuration.Add(update.RunDuration)
	}
}

func (m *Metrics) ErrorPercent(now time.Time) float64 {
	var errPct float64
	reqs := m.Count.Sum(now)
	errs := m.Errors.Sum(now)

	if reqs > 0 {
		errPct = (float64(errs) / float64(reqs) * 100)
	}

	return errPct
}

func (m *Metrics) IsHealthy(now time.Time) bool {
	return m.ErrorPercent(now) < 50
}
