package hystrix

import (
	"testing"
	"time"
)

func TestSetConcurrency(t *testing.T) {
	ConfigureCommand("conc", CommandConfig{MaxConcurrentRequests: 100})

	tickets := ConcurrentThrottle("conc")

	expected := 100
	if len(tickets) != expected {
		t.Fatalf("expected %v tickets, found %v", expected, len(tickets))
	}
}

func TestSetTimeout(t *testing.T) {
	ConfigureCommand("time", CommandConfig{Timeout: 10000})

	d := GetTimeout("time")

	expected := time.Duration(10 * time.Second)
	if d != expected {
		t.Fatalf("expected %v timeout, found %v", expected, d)
	}
}

func TestRVT(t *testing.T) {
	ConfigureCommand("rvt", CommandConfig{RequestVolumeThreshold: 30})

	rvt := GetRequestVolumeThreshold("rvt")

	expected := uint64(30)
	if rvt != expected {
		t.Fatalf("expected threshold of %v, found %v", expected, rvt)
	}
}

func TestSleepWindowDefault(t *testing.T) {
	ConfigureCommand("sleep", CommandConfig{})

	sleep := GetSleepWindow("sleep")
	expected := time.Duration(5 * time.Second)

	if sleep != expected {
		t.Fatalf("expected window of %v, found %v", expected, sleep)
	}
}