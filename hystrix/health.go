package hystrix

import "time"
import "sync"

type Health struct {
	Updates chan HealthUpdate
	Buckets map[int64]*HealthBucket
	mutex   *sync.Mutex
}

func NewHealth() Health {
	h := Health{}
	h.Updates = make(chan HealthUpdate)
	h.Buckets = map[int64]*HealthBucket{}
	h.mutex = &sync.Mutex{}

	go h.Monitor()

	return h
}

func (health *Health) Monitor() {
	var healthUpdate HealthUpdate

	for {
		healthUpdate = <-health.Updates
		b, exists := health.Buckets[healthUpdate.ts.Unix()]
		if !exists {
			health.Buckets[healthUpdate.ts.Unix()] = &HealthBucket{}
			b = health.Buckets[healthUpdate.ts.Unix()]
		}
		health.mutex.Lock()
		if healthUpdate.success {
			b.num_success += 1
		} else {
			b.num_failure += 1
		}
		health.mutex.Unlock()

		// TODO: clean old map entries
	}
}

func (health *Health) IsHealthy() bool {
	// TODO: have this based on recent_healths
	successes := 0
	failures := 0

	health.mutex.Lock()
	defer health.mutex.Unlock()
	for _, value := range health.Buckets {
		successes += value.num_success
		failures += value.num_failure
	}

	return failures <= successes
}

type HealthUpdate struct {
	success bool
	ts      time.Time
}

type HealthBucket struct {
	num_success int
	num_failure int
}
