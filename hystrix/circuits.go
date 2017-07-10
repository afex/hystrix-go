package hystrix

import (
	"log"
	"sync"
	"time"
)

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

// GetCircuit returns the circuit for the given command and whether this call created it.
func (c *Circuits) GetCircuit(name string) (*CircuitBreaker, bool, error) {
	c.BreakersMutex.RLock()
	_, ok := c.Breakers[name]
	if !ok {
		c.BreakersMutex.RUnlock()
		c.BreakersMutex.Lock()
		defer c.BreakersMutex.Unlock()
		// because we released the rlock before we obtained the exclusive lock,
		// we need to double check that some other thread didn't beat us to
		// creation.
		if cb, ok := c.Breakers[name]; ok {
			return cb, false, nil
		}
		c.Breakers[name] = newCircuitBreaker(name, c.Settings)
	} else {
		defer c.BreakersMutex.RUnlock()
	}

	return c.Breakers[name], !ok, nil
}

// Flush purges all circuit and metric information from memory.
func (c *Circuits) Flush() {
	c.BreakersMutex.Lock()
	defer c.BreakersMutex.Unlock()

	for name, cb := range c.Breakers {
		cb.metrics.Reset()
		cb.executorPool.Metrics.Reset()
		delete(c.Breakers, name)
	}
}

// Go runs your function while tracking the health of previous calls to it.
// If your function begins slowing down or failing repeatedly, we will block
// new calls to it for you to give the dependent service time to repair.
//
// Define a fallback function if you want to define some code to execute during outages.
func (c *Circuits) Go(name string, run runFunc, fallback fallbackFunc) chan error {
	cmd := &command{
		run:          run,
		fallback:     fallback,
		start:        time.Now(),
		errChan:      make(chan error, 1),
		finished:     make(chan bool, 1),
		fallbackOnce: &sync.Once{},
	}

	// dont have methods with explicit params and returns
	// let data come in and out naturally, like with any closure
	// explicit error return to give place for us to kill switch the operation (fallback)

	circuit, _, err := c.GetCircuit(name)
	if err != nil {
		cmd.errChan <- err
		return cmd.errChan
	}
	cmd.circuit = circuit

	go func() {
		defer func() { cmd.finished <- true }()

		// Circuits get opened when recent executions have shown to have a high error rate.
		// Rejecting new executions allows backends to recover, and the circuit will allow
		// new traffic when it feels a healthly state has returned.
		if !cmd.circuit.AllowRequest() {
			cmd.errorWithFallback(ErrCircuitOpen)
			return
		}

		// As backends falter, requests take longer but don't always fail.
		//
		// When requests slow down but the incoming rate of requests stays the same, you have to
		// run more at a time to keep up. By controlling concurrency during these situations, you can
		// shed load which accumulates due to the increasing ratio of active commands to incoming requests.
		cmd.Lock()
		select {
		case cmd.ticket = <-circuit.executorPool.Tickets:
			cmd.Unlock()
		default:
			cmd.Unlock()
			cmd.errorWithFallback(ErrMaxConcurrency)
			return
		}

		runStart := time.Now()
		runErr := run()

		if !cmd.isTimedOut() {
			cmd.runDuration = time.Since(runStart)

			if runErr != nil {
				cmd.errorWithFallback(runErr)
				return
			}

			cmd.reportEvent("success")
		}
	}()

	go func() {
		defer func() {
			cmd.Lock()
			cmd.circuit.executorPool.Return(cmd.ticket)
			cmd.Unlock()

			err := cmd.circuit.ReportEvent(cmd.events, cmd.start, cmd.runDuration)
			if err != nil {
				log.Print(err)
			}
		}()

		timer := time.NewTimer(c.getSettings(name).Timeout)
		defer timer.Stop()

		select {
		case <-cmd.finished:
		case <-timer.C:
			cmd.Lock()
			cmd.timedOut = true
			cmd.Unlock()

			cmd.errorWithFallback(ErrTimeout)
			return
		}
	}()

	return cmd.errChan
}

// Do runs your function in a synchronous manner, blocking until either your function succeeds
// or an error is returned, including hystrix circuit errors
func (c *Circuits) Do(name string, run runFunc, fallback fallbackFunc) error {
	done := make(chan struct{}, 1)

	r := func() error {
		err := run()
		if err != nil {
			return err
		}

		done <- struct{}{}
		return nil
	}

	f := func(e error) error {
		err := fallback(e)
		if err != nil {
			return err
		}

		done <- struct{}{}
		return nil
	}

	var errChan chan error
	if fallback == nil {
		errChan = c.Go(name, r, nil)
	} else {
		errChan = c.Go(name, r, f)
	}

	select {
	case <-done:
		return nil
	case err := <-errChan:
		return err
	}
}

func (c *Circuits) getSettings(name string) *Settings {
	return c.Settings.getSettings(name)
}
