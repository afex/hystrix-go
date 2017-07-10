package hystrix

type runFunc func() error
type fallbackFunc func(error) error

// A CircuitError is an error which models various failure states of execution,
// such as the circuit being open or a timeout.
type CircuitError struct {
	Message string
}

func (e CircuitError) Error() string {
	return "hystrix: " + e.Message
}

var (
	// ErrMaxConcurrency occurs when too many of the same named command are executed at the same time.
	ErrMaxConcurrency = CircuitError{Message: "max concurrency"}
	// ErrCircuitOpen returns when an execution attempt "short circuits". This happens due to the circuit being measured as unhealthy.
	ErrCircuitOpen = CircuitError{Message: "circuit open"}
	// ErrTimeout occurs when the provided function takes too long to execute.
	ErrTimeout = CircuitError{Message: "timeout"}

	defaultCircuits *Circuits
)

func init() {
	defaultCircuits = NewCircuits()
}

// Go runs your function while tracking the health of previous calls to it.
// If your function begins slowing down or failing repeatedly, we will block
// new calls to it for you to give the dependent service time to repair.
//
// Define a fallback function if you want to define some code to execute during outages.
func Go(name string, run runFunc, fallback fallbackFunc) chan error {
	return defaultCircuits.Go(name, run, fallback)
}

// Do runs your function in a synchronous manner, blocking until either your function succeeds
// or an error is returned, including hystrix circuit errors
func Do(name string, run runFunc, fallback fallbackFunc) error {
	return defaultCircuits.Do(name, run, fallback)
}

// GetCircuit returns the circuit for the given command and whether this call created it.
func GetCircuit(name string) (*CircuitBreaker, bool, error) {
	return defaultCircuits.GetCircuit(name)
}

// Flush purges all circuit and metric information from memory.
func Flush() {
	defaultCircuits.Flush()
}

// Configure applies settings for a set of circuits
func Configure(cmds map[string]CommandConfig) {
	defaultCircuits.Configure(cmds)
}

// ConfigureCommand applies settings for a circuit
func ConfigureCommand(name string, config CommandConfig) {
	defaultCircuits.ConfigureCommand(name, config)
}

func GetCircuitSettings() map[string]*Settings {
	return defaultCircuits.GetCircuitSettings()
}

func getSettings(name string) *Settings {
	return defaultCircuits.getSettings(name)
}
