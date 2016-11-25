package rolling

import (
	"sync"
	"time"
)

// Number tracks a numberBucket over a bounded number of
// time buckets. Currently the buckets are one second long and only the last 10 seconds are kept.
type Number struct {
	Buckets map[int64]*numberBucket
	Mutex   *sync.RWMutex
	Rolling time.Duration // how long, in seconds, to gather the metrics
}

type numberBucket struct {
	Value float64
}

// NewNumber initializes a RollingNumber struct.
func NewNumber(rolling time.Duration) *Number {
	r := &Number{
		Buckets: make(map[int64]*numberBucket),
		Mutex:   &sync.RWMutex{},
		Rolling: rolling,
	}
	return r
}

func (r *Number) getCurrentBucket() *numberBucket {
	now := time.Now().Unix()
	var bucket *numberBucket
	var ok bool

	if bucket, ok = r.Buckets[now]; !ok {
		bucket = &numberBucket{}
		r.Buckets[now] = bucket
	}

	return bucket
}

func (r *Number) removeOldBuckets() {
	now := time.Now()
	for timestamp := range r.Buckets {
		if now.Sub(time.Unix(timestamp, 0)) >= r.Rolling {
			delete(r.Buckets, timestamp)
		}
	}
}

// Increment increments the number in current timeBucket.
func (r *Number) Increment(i float64) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	b := r.getCurrentBucket()
	b.Value += i
	r.removeOldBuckets()
}

// UpdateMax updates the maximum value in the current bucket.
func (r *Number) UpdateMax(n float64) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	b := r.getCurrentBucket()
	if n > b.Value {
		b.Value = n
	}
	r.removeOldBuckets()
}

// Sum sums the values over the buckets in the last `rolling` seconds.
func (r *Number) Sum(now time.Time) float64 {
	sum := float64(0)

	r.Mutex.RLock()
	defer r.Mutex.RUnlock()

	for timestamp, bucket := range r.Buckets {
		timespan := now.Sub(time.Unix(timestamp, 0))
		if timespan >= 0 && timespan <= r.Rolling {
			sum += bucket.Value
		}
	}

	return sum
}

// Max returns the maximum value seen in the last `rolling`` seconds.
func (r *Number) Max(now time.Time) float64 {
	var max float64

	r.Mutex.RLock()
	defer r.Mutex.RUnlock()

	for timestamp, bucket := range r.Buckets {
		timespan := now.Sub(time.Unix(timestamp, 0))
		if timespan >= 0 && timespan <= r.Rolling {
			if bucket.Value > max {
				max = bucket.Value
			}
		}
	}

	return max
}

// Avg calculates the average requests in the last `rolling` seconds.
func (r *Number) Avg(now time.Time) float64 {
	return r.Sum(now) / r.Rolling.Seconds()
}
