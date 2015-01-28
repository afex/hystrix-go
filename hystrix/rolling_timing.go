package hystrix

import (
	"math"
	"sort"
	"sync"
	"time"
)

type rollingTiming struct {
	Buckets map[int64]*timingBucket
	Mutex   *sync.RWMutex

	CachedSortedDurations byDuration
	LastCachedTime        int64
}

type timingBucket struct {
	Durations []time.Duration
}

func newRollingTiming() *rollingTiming {
	r := &rollingTiming{
		Buckets: make(map[int64]*timingBucket),
		Mutex:   &sync.RWMutex{},
	}
	return r
}

type byDuration []time.Duration

func (c byDuration) Len() int           { return len(c) }
func (c byDuration) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c byDuration) Less(i, j int) bool { return c[i] < c[j] }

func (r *rollingTiming) SortedDurations() byDuration {
	r.Mutex.RLock()
	t := r.LastCachedTime
	r.Mutex.RUnlock()

	if t+time.Duration(1*time.Second).Nanoseconds() > time.Now().UnixNano() {
		// don't recalculate if current cache is still fresh
		return r.CachedSortedDurations
	}

	var durations byDuration
	now := time.Now()

	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	for timestamp, b := range r.Buckets {
		// TODO: configurable rolling window
		if timestamp >= now.Unix()-60 {
			for _, d := range b.Durations {
				durations = append(durations, d)
			}
		}
	}

	sort.Sort(durations)

	r.CachedSortedDurations = durations
	r.LastCachedTime = time.Now().UnixNano()

	return r.CachedSortedDurations
}

func (r *rollingTiming) getCurrentBucket() *timingBucket {
	r.Mutex.RLock()
	now := time.Now()
	bucket, exists := r.Buckets[now.Unix()]
	r.Mutex.RUnlock()

	if !exists {
		r.Mutex.Lock()
		defer r.Mutex.Unlock()

		r.Buckets[now.Unix()] = &timingBucket{}
		bucket = r.Buckets[now.Unix()]
	}

	return bucket
}

func (r *rollingTiming) removeOldBuckets() {
	now := time.Now()

	for timestamp := range r.Buckets {
		// TODO: configurable rolling window
		if timestamp <= now.Unix()-60 {
			delete(r.Buckets, timestamp)
		}
	}
}

func (r *rollingTiming) Add(duration time.Duration) {
	b := r.getCurrentBucket()

	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	b.Durations = append(b.Durations, duration)
	r.removeOldBuckets()
}

func (r *rollingTiming) Percentile(p float64) uint32 {
	sortedDurations := r.SortedDurations()
	length := len(sortedDurations)
	if length <= 0 {
		return 0
	}

	pos := r.ordinal(len(sortedDurations), p) - 1
	return uint32(sortedDurations[pos].Nanoseconds() / 1000000)
}

func (r *rollingTiming) ordinal(length int, percentile float64) int64 {
	if percentile == 0 && length > 0 {
		return 1
	}

	return int64(math.Ceil((percentile / float64(100)) * float64(length)))
}

func (r *rollingTiming) Mean() uint32 {
	sortedDurations := r.SortedDurations()
	var sum time.Duration
	for _, d := range sortedDurations {
		sum += d
	}

	length := int64(len(sortedDurations))
	if length == 0 {
		return 0
	}

	return uint32(sum.Nanoseconds()/length) / 1000000
}

func (r *rollingTiming) Timings() streamCmdLatency {
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
