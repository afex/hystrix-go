package hystrix

import "time"

type RollingNumber struct {
	Buckets map[int64]*NumberBucket
}

type NumberBucket struct {
	Value uint64
}

func (r *RollingNumber) getCurrentBucket() *NumberBucket {
	now := time.Now()
	bucket, exists := r.Buckets[now.Unix()]
	if !exists {
		r.Buckets[now.Unix()] = &NumberBucket{}
		bucket = r.Buckets[now.Unix()]
	}
	return bucket
}

func (r *RollingNumber) sumValues() uint64 {
	sum := uint64(0)
	now := time.Now()

	for timestamp, bucket := range r.Buckets {
		if timestamp >= now.Unix()-10 {
			sum += bucket.Value
		}
	}

	return sum
}

func NewRollingNumber() *RollingNumber {
	r := &RollingNumber{
		Buckets: make(map[int64]*NumberBucket),
	}
	return r
}

func (r *RollingNumber) Increment() {
	b := r.getCurrentBucket()
	b.Value += 1
}

func (r *RollingNumber) Sum() uint64 {
	return r.sumValues()
}
