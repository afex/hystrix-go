package hystrix

import "testing"

func TestCachedValue(t *testing.T) {
	future := Future{ValueChannel: make(chan Result, 1)}
	future.ValueChannel <- Result{1, nil}
	if v := future.Value(); v.Result.(int) != 1 {
		t.Fail()
	}
	// we should be able to call value again and get the same answer back
	if v := future.Value(); v.Result.(int) != 1 {
		t.Fail()
	}
}
