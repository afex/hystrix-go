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

func (m *Metrics) RequestCount() uint64 {
	return m.Count.Sum()
}

func (m *Metrics) ErrorCount() uint64 {
	return m.Errors.Sum()
}
