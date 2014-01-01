package hystrix

import "time"

type Health struct {
	Updates chan HealthUpdate
	Buckets map[int64]*HealthBucket
}

func NewHealth() Health {
	h := Health{}
	h.Updates = make(chan HealthUpdate)
	h.Buckets = map[int64]*HealthBucket{}

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
		Update(b, healthUpdate)
		// TODO: clean old map entries
	}
}

func (health *Health) ForceUnhealthy() {

}

func (health *Health) IsHealthy() bool {
	// TODO: have this based on recent_healths
	successes := 0
	failures := 0

	// return circuit.health != 0
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

func Update(bucket *HealthBucket, health HealthUpdate) {
	if health.success {
		bucket.num_success += 1
	} else {
		bucket.num_failure += 1
	}
}
