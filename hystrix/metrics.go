package hystrix

import (
	"math/rand"
	"time"
)

// CommandMetrics implementations track metrics for commands.
// These metrics are used to determine the health of a command as well
// as for monitoring.
type CommandMetrics interface {
	RequestCount() uint32
	ErrorCount() uint32

	Timing0() time.Duration
	Timing25() time.Duration
	Timing50() time.Duration
	Timing75() time.Duration
	Timing90() time.Duration
	Timing95() time.Duration
	Timing99() time.Duration
	Timing995() time.Duration
	Timing100() time.Duration
}

type testCmdMetrics struct{}

var _ CommandMetrics = (*testCmdMetrics)(nil)

func (m *testCmdMetrics) RequestCount() uint32 {
	return rand.Uint32()
}

func (m *testCmdMetrics) ErrorCount() uint32 {
	return rand.Uint32()
}

func (m *testCmdMetrics) Timing0() time.Duration {
	return time.Duration(rand.Float32() * 10000)
}

func (m *testCmdMetrics) Timing25() time.Duration {
	return time.Duration(rand.Float32() * 10000)
}

func (m *testCmdMetrics) Timing50() time.Duration {
	return time.Duration(rand.Float32() * 10000)
}

func (m *testCmdMetrics) Timing75() time.Duration {
	return time.Duration(rand.Float32() * 10000)
}

func (m *testCmdMetrics) Timing90() time.Duration {
	return time.Duration(rand.Float32() * 10000)
}

func (m *testCmdMetrics) Timing95() time.Duration {
	return time.Duration(rand.Float32() * 10000)
}

func (m *testCmdMetrics) Timing99() time.Duration {
	return time.Duration(rand.Float32() * 10000)
}

func (m *testCmdMetrics) Timing995() time.Duration {
	return time.Duration(rand.Float32() * 10000)
}

func (m *testCmdMetrics) Timing100() time.Duration {
	return time.Duration(rand.Float32() * 10000)
}
