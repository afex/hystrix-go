package hystrix

import (
	"sync"
	"time"
)

var (
	// DefaultTimeout is how long to wait for command to complete, in milliseconds
	DefaultTimeout = 2000
	// DefaultMaxConcurrent is how many commands of the same type can run at the same time
	DefaultMaxConcurrent = 100
	// DefaultVolumeThreshold is the minimum number of requests needed before a circuit can be tripped due to health
	DefaultVolumeThreshold = 20
	// DefaultSleepWindow is how long, in milliseconds, to wait after a circuit opens before testing for recovery
	DefaultSleepWindow = 5000
	// DefaultErrorPercentThreshold causes circuits to open once the rolling measure of errors exceeds this percent of requests
	DefaultErrorPercentThreshold = 50
)

type Settings struct {
	Timeout                time.Duration
	MaxConcurrentRequests  int
	RequestVolumeThreshold uint64
	SleepWindow            time.Duration
	ErrorPercentThreshold  int
}

// CommandConfig is used to tune circuit settings at runtime
type CommandConfig struct {
	Timeout                int `json:"timeout"`
	MaxConcurrentRequests  int `json:"max_concurrent_requests"`
	RequestVolumeThreshold int `json:"request_volume_threshold"`
	SleepWindow            int `json:"sleep_window"`
	ErrorPercentThreshold  int `json:"error_percent_threshold"`
}

var circuitSettings map[string]*Settings
var settingsMutex *sync.RWMutex
var globalSettings = &Settings{
	Timeout:                time.Duration(DefaultTimeout) * time.Millisecond,
	MaxConcurrentRequests:  DefaultMaxConcurrent,
	RequestVolumeThreshold: uint64(DefaultVolumeThreshold),
	SleepWindow:            time.Duration(DefaultSleepWindow) * time.Millisecond,
	ErrorPercentThreshold:  DefaultErrorPercentThreshold,
}

func ConfigureGlobal(config *CommandConfig) {
	settingsMutex.Lock()
	defer settingsMutex.Unlock()

	if config.Timeout != 0 {
		globalSettings.Timeout = time.Duration(config.Timeout) * time.Millisecond
	}

	if config.MaxConcurrentRequests != 0 {
		globalSettings.MaxConcurrentRequests = config.MaxConcurrentRequests
	}

	if config.RequestVolumeThreshold != 0 {
		globalSettings.RequestVolumeThreshold = uint64(config.RequestVolumeThreshold)
	}

	if config.SleepWindow != 0 {
		globalSettings.SleepWindow = time.Duration(config.SleepWindow) * time.Millisecond
	}

	if config.ErrorPercentThreshold != 0 {
		globalSettings.ErrorPercentThreshold = config.ErrorPercentThreshold
	}
}

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

	timeout := globalSettings.Timeout
	if config.Timeout != 0 {
		timeout = time.Duration(config.Timeout) * time.Millisecond
	}

	max := globalSettings.MaxConcurrentRequests
	if config.MaxConcurrentRequests != 0 {
		max = config.MaxConcurrentRequests
	}

	volume := globalSettings.RequestVolumeThreshold
	if config.RequestVolumeThreshold != 0 {
		volume = uint64(config.RequestVolumeThreshold)
	}

	sleep := globalSettings.SleepWindow
	if config.SleepWindow != 0 {
		sleep = time.Duration(config.SleepWindow) * time.Millisecond
	}

	errorPercent := globalSettings.ErrorPercentThreshold
	if config.ErrorPercentThreshold != 0 {
		errorPercent = config.ErrorPercentThreshold
	}

	circuitSettings[name] = &Settings{
		Timeout:                timeout,
		MaxConcurrentRequests:  max,
		RequestVolumeThreshold: volume,
		SleepWindow:            sleep,
		ErrorPercentThreshold:  errorPercent,
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

func GetCircuitSettings() map[string]*Settings {
	copy := make(map[string]*Settings)

	settingsMutex.RLock()
	for key, val := range circuitSettings {
		copy[key] = val
	}
	settingsMutex.RUnlock()

	return copy
}
