package hystrix

import "testing"
import "time"
import "errors"

type GoodCommand struct{}

func (c *GoodCommand) Run() (interface{}, error) {
	return 1, nil
}
func (c *GoodCommand) Fallback(err error) (interface{}, error) {
	return 2, nil
}
func (c *GoodCommand) PoolName() string       { return "GoodCommand" }
func (c *GoodCommand) Timeout() time.Duration { return time.Millisecond * 100 }

func TestExecute(t *testing.T) {
	resultChan := make(chan int)
	errChan := Go("good", func() error {
		resultChan <- 1
		return nil
	}, nil)

	select {
		case result := <-resultChan:
			if result != 1 {
				t.Fail()
			}
		case _ = <-errChan:
			t.Fail()
	}	
}

func TestQueue(t *testing.T) {
	command := NewCommand(&GoodCommand{})
	results, _ := command.Queue()
	if r := <-results; r != 1 {
		t.Fail()
	}
}

type BadCommand struct{}

func (c *BadCommand) Run() (interface{}, error) {
	return nil, errors.New("sup")
}
func (c *BadCommand) Fallback(err error) (interface{}, error) {
	return 1, nil
}
func (c *BadCommand) PoolName() string       { return "BadCommand" }
func (c *BadCommand) Timeout() time.Duration { return time.Millisecond * 100 }

func TestFallback(t *testing.T) {
	command := NewCommand(&BadCommand{})
	result, _ := command.Execute()
	if result != 1 {
		t.Fail()
	}
}

type SlowCommand struct{}

func (c *SlowCommand) Run() (interface{}, error) {
	time.Sleep(1 * time.Second)
	return 2, nil
}
func (c *SlowCommand) Fallback(err error) (interface{}, error) {
	return 1, nil
}
func (c *SlowCommand) PoolName() string       { return "SlowCommand" }
func (c *SlowCommand) Timeout() time.Duration { return time.Millisecond * 100 }

// TODO: how can we be sure the fallback is triggered from timeout.  error type?
func TestTimeout(t *testing.T) {
	command := NewCommand(&SlowCommand{})
	result, _ := command.Execute()
	if result != 1 {
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
	result, _ := command3.Execute()

	if result != 2 {
		t.Fail()
	}
}

func TestOpenCircuit(t *testing.T) {
	command := NewCommand(&GoodCommand{})
	command.ExecutorPool.Circuit.ForceOpen = true
	result, _ := command.Execute()
	if result == 1 {
		t.Fail()
	}

	// BUG: the executor pool is not naturally reset between tests
	executorPools = make(map[string]*ExecutorPool)
}
