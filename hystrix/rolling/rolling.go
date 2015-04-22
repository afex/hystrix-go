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
}

type numberBucket struct {
	Value uint64
}

// NewNumber initializes a RollingNumber struct.
func NewNumber() *Number {
	r := &Number{
		Buckets: make(map[int64]*numberBucket),
		Mutex:   &sync.RWMutex{},
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
	now := time.Now().Unix() - 10

	for timestamp := range r.Buckets {
		// TODO: configurable rolling window
		if timestamp <= now {
			delete(r.Buckets, timestamp)
		}
	}
}

// Increment increments the number in current timeBucket.
func (r *Number) Increment() {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	b := r.getCurrentBucket()
	b.Value++
	r.removeOldBuckets()
}

// UpdateMax updates the maximum value in the current bucket.
func (r *Number) UpdateMax(n int) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	b := r.getCurrentBucket()
	if uint64(n) > b.Value {
		b.Value = uint64(n)
	}
	r.removeOldBuckets()
}

// Sum sums the values over the buckets in the last 10 seconds.
func (r *Number) Sum(now time.Time) uint64 {
	sum := uint64(0)

	r.Mutex.RLock()
	defer r.Mutex.RUnlock()

	for timestamp, bucket := range r.Buckets {
		// TODO: configurable rolling window
		if timestamp >= now.Unix()-10 {
			sum += bucket.Value
		}
	}

	return sum
}

// Max returns the maximum value seen in the last 10 seconds.
func (r *Number) Max(now time.Time) uint64 {
	var max uint64

	r.Mutex.RLock()
	defer r.Mutex.RUnlock()

	for timestamp, bucket := range r.Buckets {
		// TODO: configurable rolling window
		if timestamp >= now.Unix()-10 {
			if bucket.Value > max {
				max = bucket.Value
			}
		}
	}

	return max
}
