package hystrix

import "time"

type RollingNumber struct {
	value uint32
}

func (r *RollingNumber) Increment() {
	r.value += 1
}

func (r *RollingNumber) Sum() uint32 {
	return r.value
}

type RollingPercentile struct {
	value time.Duration
	Buckets map[int64]*Bucket
}

func NewRollingPercentile() *RollingPercentile {
	r := &RollingPercentile{}
	r.Buckets = make(map[int64]*Bucket)
	return r
}

func (r *RollingPercentile) Add(duration time.Duration) {
	
}

func (r *RollingPercentile) Percentile(p float32) uint32 {
	return uint32(r.value.Nanoseconds() / 1000000)
}

func (r *RollingPercentile) Mean() time.Duration {
	return r.value
}

func (r *RollingPercentile) Timings() streamCmdLatency {
	return streamCmdLatency{
		Timing0:   r.Percentile(0),
		Timing25:  r.Percentile(25),
		Timing50:  r.Percentile(50),
		Timing75:  r.Percentile(75),
		Timing90:  r.Percentile(90),
		Timing95:  r.Percentile(95),
		Timing99:  r.Percentile(99),
		Timing995: r.Percentile(99.5),
		Timing100: r.Percentile(100),
	}
}
