package hystrix

import (
	"sync"
	"time"
)

type rollingNumber struct {
	Buckets map[int64]*numberBucket
	Mutex   *sync.RWMutex
}

type numberBucket struct {
	Value uint64
}

func newRollingNumber() *rollingNumber {
	r := &rollingNumber{
		Buckets: make(map[int64]*numberBucket),
		Mutex:   &sync.RWMutex{},
	}
	return r
}

func (r *rollingNumber) getCurrentBucket() *numberBucket {
	r.Mutex.RLock()

	now := time.Now()
	bucket, exists := r.Buckets[now.Unix()]
	if !exists {
		r.Mutex.RUnlock()
		r.Mutex.Lock()
		defer r.Mutex.Unlock()

		r.Buckets[now.Unix()] = &numberBucket{}
		bucket = r.Buckets[now.Unix()]
	} else {
		defer r.Mutex.RUnlock()
	}
	return bucket
}

func (r *rollingNumber) removeOldBuckets() {
	now := time.Now()

	for timestamp := range r.Buckets {
		// TODO: configurable rolling window
		if timestamp <= now.Unix()-10 {
			delete(r.Buckets, timestamp)
		}
	}
}

func (r *rollingNumber) Increment() {
	b := r.getCurrentBucket()

	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	b.Value++
	r.removeOldBuckets()
}

func (r *rollingNumber) UpdateMax(n int) {
	b := r.getCurrentBucket()

	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	if uint64(n) > b.Value {
		b.Value = uint64(n)
	}
	r.removeOldBuckets()
}

func (r *rollingNumber) Sum(now time.Time) uint64 {
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

func (r *rollingNumber) Max(now time.Time) uint64 {
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
