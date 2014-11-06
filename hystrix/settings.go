package hystrix

import "time"

// timeoutForCommand should read from a structure in memory to allow the
// containing application to configure known commands at run-time.
func timeoutForCommand(name string) time.Duration {
	return time.Second * 10
}