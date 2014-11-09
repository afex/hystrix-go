package hystrix

import "time"

// Health represents the last 10 seconds of Updates, tracking the ratio of success to failure.
type Health struct {
	Updates chan bool
	Buckets map[int64]*Bucket
}

// Bucket represents the success and failure rate for a given second.
type Bucket struct {
	success int
	failure int
}

// NewHealth creates a channel for Updates and monitors it.
func NewHealth() *Health {
	h := Health{}
	h.Updates = make(chan bool)
	h.Buckets = map[int64]*Bucket{}

	go h.Monitor()

	return &h
}

// Monitor subscribes to Updates as they are sent after command execution, and updates the success ratio for the given second.
func (health *Health) Monitor() {
	for update := range health.Updates {
		now := time.Now()

		b, exists := health.Buckets[now.Unix()]
		if !exists {
			health.Buckets[now.Unix()] = &Bucket{}
			b = health.Buckets[now.Unix()]
		}

		if update {
			b.success++
		} else {
			b.failure++
		}

		for timestamp, _ := range health.Buckets {
			if timestamp <= now.Unix()-10 {
				delete(health.Buckets, timestamp)
			}
		}
	}
}

// IsHealthy scans over the last 10 seconds of Updates and returns whether or not enough failures have happened to consider it "unhealthy".
func (health *Health) IsHealthy(now time.Time) bool {
	successes := 0
	failures := 0

	for timestamp, bucket := range health.Buckets {
		if timestamp >= now.Unix()-10 {
			successes += bucket.success
			failures += bucket.failure
		}
	}

	return failures <= successes
}
