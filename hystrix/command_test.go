package hystrix

import "testing"
import "time"
import "errors"

type GoodCommand struct {}
func (t *GoodCommand) Run(result_channel chan Result) {
	result_channel <- Result{Result: 1}
}
func (t *GoodCommand) Fallback(err error, result_channel chan Result) {
	result_channel <- Result{Result: 2} 
}

func TestExecute(t *testing.T) {
	command := NewCommand(&GoodCommand{})
	result := command.Execute()
	if result.Result != 1 {
		t.Fail()
	}
}

func TestQueue(t *testing.T) {
	command := NewCommand(&GoodCommand{})
	channel := command.Queue()
	if r := <-channel; r.Result != 1 {
		t.Fail()
	}
}

type BadCommand struct {}
func (c *BadCommand) Run(result_channel chan Result) {
	result_channel <- Result{Error: errors.New("sup")}
}
func (c *BadCommand) Fallback(err error, result_channel chan Result) {
	result_channel <- Result{Result: 1}
}

func TestFallback(t *testing.T) {
	command := NewCommand(&BadCommand{})
	result := command.Execute()
	if result.Result != 1 {
		t.Fail()
	}
}

type SlowCommand struct {}
func (c *SlowCommand) Run(result_channel chan Result) {
	time.Sleep(1 * time.Second)
	result_channel <- Result{Result: 2}
}
func (c *SlowCommand) Fallback(err error, result_channel chan Result) {
	result_channel <- Result{Result: 1}
}

// TODO: how can we be sure the fallback is triggered from timeout.  error type?
func TestTimeout(t *testing.T) {
	command := NewCommand(&SlowCommand{})
	result := command.Execute()
	if result.Result != 1 {
		t.Fail()
	}
}

// TODO: how can we be sure the fallback is triggered from full pool.  error type?
func TestFullExecutorPool(t *testing.T) {
	pool := NewExecutorPool("TestFullExecutorPool", 2)

	command1 := NewCommand(&SlowCommand{})
	command1.ExecutorPool = pool
	command2 := NewCommand(&SlowCommand{})
	command2.ExecutorPool = pool
	command3 := NewCommand(&GoodCommand{})
	command3.ExecutorPool = pool

	command1.Queue()
	command2.Queue()
	result := command3.Execute()

	if result.Result != 2 {
		t.Fail()
	}
}

func TestOpenCircuit(t *testing.T) {
	command := NewCommand(&GoodCommand{})
	command.ExecutorPool.Circuit.ForceOpen = true
	result := command.Execute()
	if result.Result == 1 {
		t.Fail()
	}

	// BUG: the executor pool is not naturally reset between tests
	executorPools = make(map[string]*ExecutorPool)
}
