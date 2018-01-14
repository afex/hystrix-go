package hystrix

import (
	"sync"
	"time"
)

var (
	// DefaultTimeout is how long to wait for command to complete, in milliseconds
	DefaultTimeout = 1000
	// DefaultMaxConcurrent is how many commands of the same type can run at the same time
	DefaultMaxConcurrent = 10
	// DefaultVolumeThreshold is the minimum number of requests needed before a circuit can be tripped due to health
	DefaultVolumeThreshold = 20
	// DefaultSleepWindow is how long, in milliseconds, to wait after a circuit opens before testing for recovery
	DefaultSleepWindow = 5000
	// DefaultErrorPercentThreshold causes circuits to open once the rolling measure of errors exceeds this percent of requests
	DefaultErrorPercentThreshold = 50
)

// Settings Setting for the hystrixCommand
type Settings struct {
	Timeout                     time.Duration
	MaxConcurrentRequests       int
	RequestVolumeThreshold      uint64
	SleepWindow                 time.Duration
	ErrorPercentThreshold       int
	QueueSizeRejectionThreshold int
}

// CommandConfig is used to tune circuit settings at runtime
type CommandConfig struct {
	Timeout                int `json:"timeout"`
	MaxConcurrentRequests  int `json:"max_concurrent_requests"`
	RequestVolumeThreshold int `json:"request_volume_threshold"`
	SleepWindow            int `json:"sleep_window"`
	ErrorPercentThreshold  int `json:"error_percent_threshold"`
	// for more details refer - https://github.com/Netflix/Hystrix/wiki/Configuration#maxqueuesize
	QueueSizeRejectionThreshold int `json:"queue_size_rejection_threshold"`
}

var circuitSettings map[string]*Settings
var settingsMutex *sync.RWMutex

func init() {
	circuitSettings = make(map[string]*Settings)
	settingsMutex = &sync.RWMutex{}
}

// Configure applies settings for a set of circuits
func Configure(cmds map[string]CommandConfig) {
	for k, v := range cmds {
		ConfigureCommand(k, v)
	}
}

// ConfigureCommand applies settings for a circuit
func ConfigureCommand(name string, config CommandConfig) {
	settingsMutex.Lock()
	defer settingsMutex.Unlock()

	timeout := DefaultTimeout
	if config.Timeout != 0 {
		timeout = config.Timeout
	}

	max := DefaultMaxConcurrent
	if config.MaxConcurrentRequests != 0 {
		max = config.MaxConcurrentRequests
	}

	volume := DefaultVolumeThreshold
	if config.RequestVolumeThreshold != 0 {
		volume = config.RequestVolumeThreshold
	}

	sleep := DefaultSleepWindow
	if config.SleepWindow != 0 {
		sleep = config.SleepWindow
	}

	errorPercent := DefaultErrorPercentThreshold
	if config.ErrorPercentThreshold != 0 {
		errorPercent = config.ErrorPercentThreshold
	}

	// queueSizeRejectionThreshold should ideally be a function of DefaultMaxConcurrent
	// drawing parallels with netflix/hystrix default setting, it is set equal to DefaultMaxConcurrent
	queueSizeRejectionThreshold := max
	if config.QueueSizeRejectionThreshold != 0 {
		queueSizeRejectionThreshold = config.QueueSizeRejectionThreshold
	}

	circuitSettings[name] = &Settings{
		Timeout:                     time.Duration(timeout) * time.Millisecond,
		MaxConcurrentRequests:       max,
		RequestVolumeThreshold:      uint64(volume),
		SleepWindow:                 time.Duration(sleep) * time.Millisecond,
		ErrorPercentThreshold:       errorPercent,
		QueueSizeRejectionThreshold: queueSizeRejectionThreshold,
	}
}

func getSettings(name string) *Settings {
	settingsMutex.RLock()
	s, exists := circuitSettings[name]
	settingsMutex.RUnlock()

	if !exists {
		ConfigureCommand(name, CommandConfig{})
		s = getSettings(name)
	}

	return s
}

// GetCircuitSettings Returns a copy of the hystrix circuit map
func GetCircuitSettings() map[string]*Settings {
	copy := make(map[string]*Settings)

	settingsMutex.RLock()
	for key, val := range circuitSettings {
		copy[key] = val
	}
	settingsMutex.RUnlock()

	return copy
}
