package hystrix

import (
	"sync"
	"time"
)

type RollingNumber struct {
	Buckets map[int64]*NumberBucket
	Mutex   *sync.RWMutex
}

type NumberBucket struct {
	Value uint64
}

func (r *RollingNumber) getCurrentBucket() *NumberBucket {
	r.Mutex.RLock()

	now := time.Now()
	bucket, exists := r.Buckets[now.Unix()]
	if !exists {
		r.Mutex.RUnlock()
		r.Mutex.Lock()
		defer r.Mutex.Unlock()

		r.Buckets[now.Unix()] = &NumberBucket{}
		bucket = r.Buckets[now.Unix()]
	} else {
		defer r.Mutex.RUnlock()
	}
	return bucket
}

func (r *RollingNumber) removeOldBuckets() {
	now := time.Now()

	for timestamp, _ := range r.Buckets {
		// TODO: configurable rolling window
		if timestamp <= now.Unix()-10 {
			delete(r.Buckets, timestamp)
		}
	}
}

func NewRollingNumber() *RollingNumber {
	r := &RollingNumber{
		Buckets: make(map[int64]*NumberBucket),
		Mutex:   &sync.RWMutex{},
	}
	return r
}

func (r *RollingNumber) Increment() {
	b := r.getCurrentBucket()

	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	b.Value += 1
	r.removeOldBuckets()
}

func (r *RollingNumber) Sum(now time.Time) uint64 {
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
