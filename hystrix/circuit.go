package hystrix

type CircuitBreaker struct {
	health        float32
	HealthUpdates chan int
}

func NewCircuitBreaker() *CircuitBreaker {
	c := &CircuitBreaker{}
	c.health = 0
	c.HealthUpdates = make(chan int)

	// go c.monitorHealth()

	return c
}

// func (circuit *CircuitBreaker) monitorHealth() {
// 	var health int = 0
// 	recent_healths := [10]int{}
// 	for {
// 		health = <-circuit.HealthUpdates
// 		// TODO: remove oldest from recent_healths
// 		// TODO: add newest to recent_healths
// 	}
// }

func (circuit *CircuitBreaker) IsOpen() bool {
	// TODO: have this based on recent_healths
	return circuit.health != 0
}

func (circuit *CircuitBreaker) Open() {
	circuit.health = 1
}