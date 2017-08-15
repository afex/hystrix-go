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

type SettingsCollection struct {
	Settings      map[string]*Settings
	SettingsMutex *sync.RWMutex
}

func NewSettingsCollection() *SettingsCollection {
	return &SettingsCollection{
		Settings:      make(map[string]*Settings),
		SettingsMutex: &sync.RWMutex{},
	}
}

func (c *SettingsCollection) ConfigureCommand(name string, config CommandConfig) {
	c.SettingsMutex.Lock()
	defer c.SettingsMutex.Unlock()

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

	c.Settings[name] = &Settings{
		Timeout:                time.Duration(timeout) * time.Millisecond,
		MaxConcurrentRequests:  max,
		RequestVolumeThreshold: uint64(volume),
		SleepWindow:            time.Duration(sleep) * time.Millisecond,
		ErrorPercentThreshold:  errorPercent,
	}
}

func (c *SettingsCollection) getSettings(name string) *Settings {
	c.SettingsMutex.RLock()
	s, exists := c.Settings[name]
	c.SettingsMutex.RUnlock()

	if !exists {
		c.ConfigureCommand(name, CommandConfig{})
		s = c.getSettings(name)
	}

	return s
}

func (c *SettingsCollection) GetCircuitSettings() map[string]*Settings {
	copy := make(map[string]*Settings)

	c.SettingsMutex.RLock()
	for key, val := range c.Settings {
		copy[key] = val
	}
	c.SettingsMutex.RUnlock()

	return copy
}

