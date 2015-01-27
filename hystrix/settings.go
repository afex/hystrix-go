package hystrix

import (
	"sync"
	"time"
)

const (
	DefaultTimeout               = 1000
	DefaultMaxConcurrent         = 10
	DefaultVolumeThreshold       = 20
	DefaultSleepWindow           = 5000
	DefaultErrorPercentThreshold = 50
)

type Settings struct {
	Timeout                time.Duration
	MaxConcurrentRequests  int
	RequestVolumeThreshold uint64
	SleepWindow            time.Duration
	ErrorPercentThreshold  int
}

type CommandConfig struct {
	Timeout                int `json:"timeout"`
	MaxConcurrentRequests  int `json:"max_concurrent_requests"`
	RequestVolumeThreshold int `json:"request_volume_threshold"`
	SleepWindow            int `json:"sleep_window"`
	ErrorPercentThreshold  int `json:"error_percent_threshold"`
}

var settings map[string]*Settings
var settingsMutex *sync.RWMutex

func init() {
	settings = make(map[string]*Settings)
	settingsMutex = &sync.RWMutex{}
}

func Configure(cmds map[string]CommandConfig) {
	for k, v := range cmds {
		ConfigureCommand(k, v)
	}
}

func ConfigureCommand(name string, config CommandConfig) *Settings {
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

	settings[name] = &Settings{
		Timeout:                time.Duration(timeout) * time.Millisecond,
		MaxConcurrentRequests:  max,
		RequestVolumeThreshold: uint64(volume),
		SleepWindow:            time.Duration(sleep) * time.Millisecond,
		ErrorPercentThreshold:  errorPercent,
	}

	return settings[name]
}

func getSettings(name string) *Settings {
	settingsMutex.RLock()
	s, exists := settings[name]
	settingsMutex.RUnlock()

	if !exists {
		s = ConfigureCommand(name, CommandConfig{})
	}

	return s
}

// GetTimeout returns the timeout setting for the given command.
func GetTimeout(name string) time.Duration {
	return getSettings(name).Timeout
}

func GetConcurrency(name string) int {
	return getSettings(name).MaxConcurrentRequests
}

func GetRequestVolumeThreshold(name string) uint64 {
	return getSettings(name).RequestVolumeThreshold
}

func GetSleepWindow(name string) time.Duration {
	return getSettings(name).SleepWindow
}

func GetErrorPercentThreshold(name string) int {
	return getSettings(name).ErrorPercentThreshold
}
