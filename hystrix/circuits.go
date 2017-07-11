package hystrix

import "sync"

type Circuits struct {
	Breakers      map[string]*CircuitBreaker
	BreakersMutex *sync.RWMutex
	Settings      *SettingsCollection
}

func NewCircuits() *Circuits {
	return &Circuits{
		Breakers:      make(map[string]*CircuitBreaker),
		BreakersMutex: &sync.RWMutex{},
		Settings:      NewSettingsCollection(),
	}
}

func (c *Circuits) Configure(cmds map[string]CommandConfig) {
	for k, v := range cmds {
		c.ConfigureCommand(k, v)
	}
}

func (c *Circuits) ConfigureCommand(name string, config CommandConfig) {
	c.Settings.ConfigureCommand(name, config)
}

func (c *Circuits) GetCircuitSettings() map[string]*Settings {
	return c.Settings.GetCircuitSettings()
}
