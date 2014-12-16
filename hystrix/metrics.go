package hystrix

import (
	"log"
	"time"
)

type ExecutionMetric struct {
	Type     string
	Time     time.Time
	TotalDuration time.Duration
	RunDuration time.Duration
}

type Metrics struct {
	Updates chan *ExecutionMetric

	Count  *RollingNumber
	Errors *RollingNumber

	TotalDuration *RollingPercentile
	RunDuration *RollingPercentile
}

func NewMetrics() *Metrics {
	m := &Metrics{}

	m.Updates = make(chan *ExecutionMetric)

	m.Count = &RollingNumber{}
	m.Errors = &RollingNumber{}
	m.TotalDuration = NewRollingPercentile()
	m.RunDuration = NewRollingPercentile()

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

func (m *Metrics) RequestCount() uint32 {
	return m.Count.Sum()
}

func (m *Metrics) ErrorCount() uint32 {
	return m.Errors.Sum()
}
