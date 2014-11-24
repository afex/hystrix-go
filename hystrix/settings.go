package hystrix

import "time"

// Config stores run-time command configuration
type Config struct {
	Timeouts    map[string]time.Duration
	Concurrency map[string]chan *Ticket
}

var config *Config

// Ticket is grabbed by each command before execution can start
type Ticket struct{}

func init() {
	config = &Config{
		Timeouts:    make(map[string]time.Duration),
		Concurrency: make(map[string]chan *Ticket),
	}
}

// GetTimeout returns the timeout setting for the given command.
func GetTimeout(name string) time.Duration {
	if val, ok := config.Timeouts[name]; ok {
		return val
	}

	return time.Second * 10
}

// SetTimeout changes the timeout setting for the given command, affecting
// all future runs
func SetTimeout(name string, duration time.Duration) error {
	config.Timeouts[name] = duration
	return nil
}

// SetConcurrency changes how many of a given command are allowed to run
// at the same time before tripping the fallback logic
func SetConcurrency(name string, max int) error {
	config.Concurrency[name] = make(chan *Ticket, max)
	return nil
}

// ConcurrentThrottle hands out a channel which commands use to throttle how many of
// each command can run at a time.  If a command can't pull from the channel on the first attempt
// it triggers the fallback.
func ConcurrentThrottle(name string) (chan *Ticket, error) {
	if val, ok := config.Concurrency[name]; ok {
		return val, nil
	}
	config.Concurrency[name] = make(chan *Ticket, 10)

	return config.Concurrency[name], nil
}
