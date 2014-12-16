package hystrix

import (
	"log"
	"math"
	"sort"
	"time"
)

type RollingTiming struct {
	Buckets map[int64]*TimingBucket
}

type TimingBucket struct {
	Durations []time.Duration
}

func NewRollingTiming() *RollingTiming {
	r := &RollingTiming{
		Buckets: make(map[int64]*TimingBucket),
	}
	return r
}

type ByDuration []time.Duration

func (c ByDuration) Len() int           { return len(c) }
func (c ByDuration) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c ByDuration) Less(i, j int) bool { return c[i] < c[j] }

func (r *RollingTiming) SortedDurations() ByDuration {
	var durations ByDuration

	for _, b := range r.Buckets {
		for _, d := range b.Durations {
			durations = append(durations, d)
		}
	}

	sort.Sort(durations)

	return durations
}

func (r *RollingTiming) getCurrentBucket() *TimingBucket {
	now := time.Now()
	bucket, exists := r.Buckets[now.Unix()]
	if !exists {
		r.Buckets[now.Unix()] = &TimingBucket{}
		bucket = r.Buckets[now.Unix()]
	}
	return bucket
}

func (r *RollingTiming) Add(duration time.Duration) {
	b := r.getCurrentBucket()
	b.Durations = append(b.Durations, duration)
}

func (r *RollingTiming) Percentile(p float64) uint32 {
	sortedDurations := r.SortedDurations()
	length := len(sortedDurations)
	if length <= 0 {
		return 0
	}

	pos := r.ordinal(len(sortedDurations), p) - 1
	log.Printf("len: %v perc: %v, pos: %v", len(sortedDurations), p, pos)
	return uint32(sortedDurations[pos].Nanoseconds() / 1000000)
}

func (r *RollingTiming) ordinal(length int, percentile float64) int64 {
	if length == 1 {
		return 1
	}
	
	return int64(math.Ceil((percentile / float64(100)) * float64(length)))
}

func (r *RollingTiming) Mean() int64 {
	sortedDurations := r.SortedDurations()
	var sum time.Duration
	for _, d := range sortedDurations {
		sum += d
	}

	return sum.Nanoseconds() / int64(len(sortedDurations))
}

func (r *RollingTiming) Timings() streamCmdLatency {
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
