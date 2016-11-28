package config

import (
	"sync"
	"time"
)

var (
	// DefaultEnabled is whether circuit breaker should be enabled
	DefaultEnabled = true
	// DefaultTimeout is how long to wait for command to complete, in milliseconds
	DefaultTimeout = 1000
	// DefaultForceOpen is false by default, we want allow traffic
	DefaultForceOpen = false
	// DefaultForceClosed is false by default, we want circuit break work
	DefaultForceClosed = false
	// DefaultMaxConcurrent is how many commands of the same type can run at the same time
	DefaultMaxConcurrent = 10
	// DefaultVolumeThreshold is the minimum number of requests needed before a circuit can be tripped due to health
	DefaultVolumeThreshold = 20
	// DefaultRollingWindow is how long, in seconds, to gather the metrics
	DefaultRollingWindow = 10
	// DefaultSleepWindow is how long, in milliseconds, to wait after a circuit opens before testing for recovery
	DefaultSleepWindow = 5000
	// DefaultErrorPercentThreshold causes circuits to open once the rolling measure of errors exceeds this percent of requests
	DefaultErrorPercentThreshold = 50

	// DefaultMetricsRollingStatisticalWindow is how long, in milliseconds, to gather the metrics
	DefaultMetricsRollingStatisticalWindow = 10000 * time.Millisecond
	// DefaultMetricsRollingStatisticalWindowBuckets is the number of buckets in the statisticalWindow
	DefaultMetricsRollingStatisticalWindowBuckets = 10
)

// Settings represents the hystrix's configurations to use when checking whether it's time to create a short circuit or
// recover the circuit.
type Settings struct {
	Enabled                bool          // whether circuit breaker should be enabled.
	Timeout                time.Duration // the command timeout
	ForceOpen              bool          // a property to allow forcing the circuit open
	ForceClosed            bool          // a property to allow ignoring errors and therefore never trip 'open'
	MaxConcurrentRequests  int           // max concurrent requests
	RequestVolumeThreshold uint64        // the threshold of the requests within the rolling window
	SleepWindow            time.Duration // the sleep window duration
	ErrorPercentThreshold  int           // the threshold of the error percentage

	MetricsRollingStatisticalWindow        time.Duration // the duration back that will be tracked
	MetricsRollingStatisticalWindowBuckets int           // number of buckets in the statisticalWindow
}

// CommandConfig is used to tune circuit settings at runtime
type CommandConfig struct {
	Enabled                                string `json:"circuit_breaker_enabled"`
	Timeout                                int    `json:"timeout"`
	ForceOpen                              bool   `json:"force_open"`
	ForceClosed                            bool   `json:"force_closed"`
	MaxConcurrentRequests                  int    `json:"max_concurrent_requests"`
	RequestVolumeThreshold                 int    `json:"request_volume_threshold"`
	SleepWindow                            int    `json:"sleep_window"`
	ErrorPercentThreshold                  int    `json:"error_percent_threshold"`
	MetricsRollingStatisticalWindow        int    `json:"metrics_rolling_window"`
	MetricsRollingStatisticalWindowBuckets int    `json:"metrics_rolling_buckets"`
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

	enabled := DefaultEnabled
	if config.Enabled == "false" {
		enabled = false
	} else {
		enabled = true
	}

	timeout := DefaultTimeout
	if config.Timeout != 0 {
		timeout = config.Timeout
	}

	forceOpen := DefaultForceOpen
	if config.ForceOpen != DefaultForceOpen {
		forceOpen = config.ForceOpen
	}

	forceClosed := DefaultForceClosed
	if config.ForceClosed != DefaultForceClosed {
		forceClosed = config.ForceClosed
	}

	if forceOpen == true && forceClosed == true {
		panic("forceOpen and forceClosed have conflicts")
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

	rollingBuckets := DefaultMetricsRollingStatisticalWindowBuckets
	if config.MetricsRollingStatisticalWindowBuckets != 0 {
		rollingBuckets = config.MetricsRollingStatisticalWindowBuckets
	}

	rollingWindow := DefaultMetricsRollingStatisticalWindow
	if config.MetricsRollingStatisticalWindow != 0 {
		rollingWindow = time.Duration(config.MetricsRollingStatisticalWindow) * time.Millisecond
	}

	circuitSettings[name] = &Settings{
		Enabled:                enabled,
		Timeout:                time.Duration(timeout) * time.Millisecond,
		ForceOpen:              forceOpen,
		ForceClosed:            forceClosed,
		MaxConcurrentRequests:  max,
		RequestVolumeThreshold: uint64(volume),
		SleepWindow:            time.Duration(sleep) * time.Millisecond,
		ErrorPercentThreshold:  errorPercent,

		MetricsRollingStatisticalWindow:        rollingWindow,
		MetricsRollingStatisticalWindowBuckets: rollingBuckets,
	}
}

// GetSettings returns the settings of the specified circuit breaker
func GetSettings(name string) *Settings {
	settingsMutex.RLock()
	s, exists := circuitSettings[name]
	// debug.PrintStack()
	settingsMutex.RUnlock()

	if !exists {
		ConfigureCommand(name, CommandConfig{})
		s = GetSettings(name)
	}

	return s
}

// GetCircuitSettings returns the copy of the circuit settings.
func GetCircuitSettings() map[string]*Settings {
	copy := make(map[string]*Settings)

	settingsMutex.RLock()
	for key, val := range circuitSettings {
		copy[key] = val
	}
	settingsMutex.RUnlock()

	return copy
}
