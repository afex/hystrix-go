package hystrix

import "testing"
import "errors"
import "time"

// TODO: make testing easier. better descriptions, matchers for fallbacks, etc

func TestPackageLevelExecute(t *testing.T) {
	result := Execute(
		func(result_channel chan Result) { result_channel <- Result{ Result: 1 } },
		func(err error, result_channel chan Result) { result_channel <- Result{ Error: nil } },
	)
	if result.Result != 1 {
		t.Fail()
	}
}

func TestPackageLevelQueue(t *testing.T) {
	future := Queue(
		func(result_channel chan Result) { result_channel <- Result{ Result: 1 } },
		func(err error, result_channel chan Result) { result_channel <- Result{ Error: nil } },
	)
	if future.Value().Result != 1 {
		t.Fail()
	}	
}

func TestExecute(t *testing.T) {
	command := Command{
		Run: func(result_channel chan Result) { result_channel <- Result{ Result: 1 } },
		Fallback: func(err error, result_channel chan Result) { result_channel <- Result{ Error: nil } },
	}
	result := command.Execute()
	if result.Result != 1 {
		t.Fail()
	}
}

func TestQueue(t *testing.T) {
	command := Command{
		Run: func(result_channel chan Result) { result_channel <- Result{ Result: 1 } },
		Fallback: func(err error, result_channel chan Result) { result_channel <- Result{ Error: nil } },
	}
	future := command.Queue()
	if future.Value().Result != 1 {
		t.Fail()
	}	
}

func TestFallback(t *testing.T) {
	command := Command{
		Run: func(result_channel chan Result) { result_channel <- Result{ Error: errors.New("sup") } }, 
		Fallback: func(err error, result_channel chan Result) { result_channel <- Result{ Result: 1 } },
	}
	result := command.Execute()
	if result.Result != 1 {
		t.Fail()
	}
}

// TODO: how can we be sure the fallback is triggered from timeout.  error type?
func TestTimeout(t *testing.T) {
	command := Command{
		Run: func(result_channel chan Result) { time.Sleep(1 * time.Second); result_channel <- Result{ Result: 2 } }, 
		Fallback: func(err error, result_channel chan Result) { result_channel <- Result{ Result: 1 } },
	}
	result := command.Execute()
	if result.Result != 1 {
		t.Fail()
	}
}

// TODO: how can we be sure the fallback is triggered from full pool.  error type?
func TestFullExecutorPool(t *testing.T) {
	command1 := Command{
		Run: func(result_channel chan Result) { time.Sleep(10 * time.Millisecond) }, 
		Fallback: func(err error, result_channel chan Result) { result_channel <- Result{ Result: 1 } },
		ExecutorPoolSize: 2,
		ExecutorPoolName: "TestFullExecutorPool",
	}
	command2 := Command{
		Run: func(result_channel chan Result) { time.Sleep(10 * time.Millisecond) }, 
		Fallback: func(err error, result_channel chan Result) { result_channel <- Result{ Result: 1 } },
		ExecutorPoolSize: 2,
		ExecutorPoolName: "TestFullExecutorPool",
	}
	command3 := Command{
		Run: func(result_channel chan Result) { result_channel <- Result{ Result: 2 } }, 
		Fallback: func(err error, result_channel chan Result) { result_channel <- Result{ Result: 1 } },
		ExecutorPoolSize: 2,
		ExecutorPoolName: "TestFullExecutorPool",
	}

	command1.Queue()
	command2.Queue()
	result := command3.Execute()

	if result.Result != 1 {
		t.Fail()
	}
}