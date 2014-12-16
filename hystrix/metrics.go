package hystrix

import (
	"log"
	"math/rand"
	"time"
)

type ExecutionMetric struct {
	Type     string
	Time     int64
	Duration time.Duration
}

type Metrics struct {
	Updates chan *ExecutionMetric
}

func NewMetrics() *Metrics {
	m := &Metrics{}

	m.Updates = make(chan *ExecutionMetric)

	go m.Monitor()

	return m
}

func (m *Metrics) Monitor() {
	for update := range m.Updates {
		log.Printf("%+v", *update)
		// increment rolling values
	}
}

func (m *Metrics) RequestCount() uint32 {
	return rand.Uint32()
}

func (m *Metrics) ErrorCount() uint32 {
	return rand.Uint32()
}

func (m *Metrics) Timing0() time.Duration {
	return time.Duration(rand.Float32() * 10000)
}

func (m *Metrics) Timing25() time.Duration {
	return time.Duration(rand.Float32() * 10000)
}

func (m *Metrics) Timing50() time.Duration {
	return time.Duration(rand.Float32() * 10000)
}

func (m *Metrics) Timing75() time.Duration {
	return time.Duration(rand.Float32() * 10000)
}

func (m *Metrics) Timing90() time.Duration {
	return time.Duration(rand.Float32() * 10000)
}

func (m *Metrics) Timing95() time.Duration {
	return time.Duration(rand.Float32() * 10000)
}

func (m *Metrics) Timing99() time.Duration {
	return time.Duration(rand.Float32() * 10000)
}

func (m *Metrics) Timing995() time.Duration {
	return time.Duration(rand.Float32() * 10000)
}

func (m *Metrics) Timing100() time.Duration {
	return time.Duration(rand.Float32() * 10000)
}
