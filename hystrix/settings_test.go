package hystrix

import "testing"

func TestSetConcurrency(t *testing.T) {
	SetConcurrency("conc", 100)
	
	tickets, err := ConcurrentThrottle("conc")
	if err != nil {
		t.Fatal(err)
	}

	expected := 100
	if len(tickets) != expected {
		t.Fatalf("expected %v tickets, found %v", expected, len(tickets))
	}
}
