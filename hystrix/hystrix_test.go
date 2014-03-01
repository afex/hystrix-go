package hystrix

/*import "testing"*/

// TODO: make testing easier. better descriptions, matchers for fallbacks, etc

/*func TestPackageLevelExecute(t *testing.T) {
	result := Execute(
		func(result_channel chan Result) { result_channel <- Result{Result: 1} },
		func(err error, result_channel chan Result) { result_channel <- Result{Error: nil} },
	)
	if result.Result != 1 {
		t.Fail()
	}
}

func TestPackageLevelQueue(t *testing.T) {
	channel := Queue(
		func(result_channel chan Result) { result_channel <- Result{Result: 1} },
		func(err error, result_channel chan Result) { result_channel <- Result{Error: nil} },
	)
	if r := <-channel; r.Result != 1 {
		t.Fail()
	}
}*/
