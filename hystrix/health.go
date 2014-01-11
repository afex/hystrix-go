package hystrix

import "time"
import "sync"

// Health represents the last 10 seconds of Updates, tracking the ratio of success to failure.
type Health struct {
	Updates chan Update
	Buckets map[int64]*Bucket
	mutex   *sync.Mutex
}

// Update is a success/failure message sent for every command execution.
type Update struct {
	success bool
	ts      time.Time
}

// Bucket represents the success and failure rate for a given second.
type Bucket struct {
	numSuccess int
	numFailure int
}

// NewHealth creates a channel for Updates and monitors it.
func NewHealth() Health {
	h := Health{}
	h.Updates = make(chan Update)
	h.Buckets = map[int64]*Bucket{}
	h.mutex = &sync.Mutex{}

	go h.Monitor()

	return h
}

// Monitor subscribes to Updates as they are sent after command execution, and updates the success ratio for the given second.
func (health *Health) Monitor() {
	var update Update

	for {
		update = <-health.Updates
		b, exists := health.Buckets[update.ts.Unix()]
		if !exists {
			health.Buckets[update.ts.Unix()] = &Bucket{}
			b = health.Buckets[update.ts.Unix()]
		}
		health.mutex.Lock()
		if update.success {
			b.numSuccess++
		} else {
			b.numFailure++
		}
		health.mutex.Unlock()

		// TODO: clean old map entries.  will leak until this is coded
	}
}

// IsHealthy scans over the last 10 seconds of Updates and returns whether or not enough failures have happened to consider it "unhealthy".
func (health *Health) IsHealthy() bool {
	successes := 0
	failures := 0

	health.mutex.Lock()
	defer health.mutex.Unlock()
	for timestamp, value := range health.Buckets {
		if timestamp >= time.Now().Unix()-10 {
			successes += value.numSuccess
			failures += value.numFailure
		}
	}

	return failures <= successes
}
