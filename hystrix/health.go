package hystrix

import "time"
import "sync"

type Health struct {
	Updates chan Update
	Buckets map[int64]*Bucket
	mutex   *sync.Mutex
}

type Update struct {
	success bool
	ts      time.Time
}

type Bucket struct {
	num_success int
	num_failure int
}

func NewHealth() Health {
	h := Health{}
	h.Updates = make(chan Update)
	h.Buckets = map[int64]*Bucket{}
	h.mutex = &sync.Mutex{}

	go h.Monitor()

	return h
}

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
			b.num_success += 1
		} else {
			b.num_failure += 1
		}
		health.mutex.Unlock()

		// TODO: clean old map entries.  will leak until this is coded
	}
}

func (health *Health) IsHealthy() bool {
	successes := 0
	failures := 0

	health.mutex.Lock()
	defer health.mutex.Unlock()
	for timestamp, value := range health.Buckets {
		if timestamp >= time.Now().Unix()-10 {
			successes += value.num_success
			failures += value.num_failure
		}
	}

	return failures <= successes
}
