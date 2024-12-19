package hystrix_go

import (
	"github.com/afex/hystrix-go/hystrix"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestInitHystrixCommandConfigs_MultipleConfigs(t *testing.T) {
	configs := []HystrixConfig{
		{
			CommandName:            "command1",
			Timeout:                1000,
			MaxConcurrentRequests:  10,
			RequestVolumeThreshold: 20,
			SleepWindow:            5000,
			ErrorPercentThreshold:  50,
		},
		{
			CommandName:            "command2",
			Timeout:                2000,
			MaxConcurrentRequests:  20,
			RequestVolumeThreshold: 30,
			SleepWindow:            6000,
			ErrorPercentThreshold:  60,
		},
	}

	InitHystrixCommandConfigs(configs)
	cmdConfig1 := hystrix.GetCircuitSettings()
	assert.Equal(t, time.Duration(1000)*time.Millisecond, cmdConfig1["command1"].Timeout)
	assert.Equal(t, 10, cmdConfig1["command1"].MaxConcurrentRequests)
	assert.Equal(t, uint64(20), cmdConfig1["command1"].RequestVolumeThreshold)
	assert.Equal(t, time.Duration(5000)*time.Millisecond, cmdConfig1["command1"].SleepWindow)
	assert.Equal(t, 50, cmdConfig1["command1"].ErrorPercentThreshold)

	cmdConfig2 := hystrix.GetCircuitSettings()
	assert.Equal(t, time.Duration(2000)*time.Millisecond, cmdConfig2["command2"].Timeout)
	assert.Equal(t, 20, cmdConfig2["command2"].MaxConcurrentRequests)
	assert.Equal(t, uint64(30), cmdConfig2["command2"].RequestVolumeThreshold)
	assert.Equal(t, time.Duration(6000)*time.Millisecond, cmdConfig2["command2"].SleepWindow)
	assert.Equal(t, 60, cmdConfig2["command2"].ErrorPercentThreshold)
}
