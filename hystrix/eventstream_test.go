package hystrix

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func sleepingCommand(t *testing.T, name string, duration time.Duration) {
	done := make(chan bool)
	errChan := Go(name, func() error {
		time.Sleep(duration)
		done <- true
		return nil
	}, nil)

	select {
	case _ = <-done:
		// do nothing
	case err := <-errChan:
		t.Fatal(err)
	}
}

func failingCommand(t *testing.T, name string, duration time.Duration) {
	done := make(chan bool)
	errChan := Go(name, func() error {
		time.Sleep(duration)
		return fmt.Errorf("fail")
	}, nil)

	select {
	case _ = <-done:
		t.Fatal("should not have succeeded")
	case _ = <-errChan:
		// do nothing
	}
}

type EventStreamTestServer struct {
	*httptest.Server
	EventStreamer
}

type EventStreamer interface {
	Stop()
}

func (s *EventStreamTestServer) stopTestServer() error {
	s.Close()
	s.Stop()
	FlushMetrics()

	return nil
}

func startTestServer() *EventStreamTestServer {
	hystrixStreamHandler := NewStreamHandler()
	hystrixStreamHandler.Start()
	return &EventStreamTestServer{
		httptest.NewServer(hystrixStreamHandler),
		hystrixStreamHandler,
	}
}

// grabFirstFromStream reads on the http request until we see the first
// full result printed
func grabFirstFromStream(t *testing.T, url string) streamCmdEvent {
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	// since the event stream connection doesn't ever close,
	// we only read until we have the message we want and
	// disconnect.
	buf := []byte{0}
	data := ""
	for {
		_, err := res.Body.Read(buf)
		if err != nil {
			t.Fatal(err)
		}

		data += string(buf)
		if strings.Contains(data, "\n\n") {
			break
		}
	}

	data = strings.Replace(data, "data:{", "{", 1)

	var event streamCmdEvent
	json.Unmarshal([]byte(data), &event)

	return event
}

func TestEventStream(t *testing.T) {
	server := startTestServer()
	defer server.stopTestServer()

	sleepingCommand(t, "eventstream", 100*time.Millisecond)
	sleepingCommand(t, "eventstream", 100*time.Millisecond)
	event := grabFirstFromStream(t, server.URL)

	if event.Name != "eventstream" {
		t.Errorf("expected name to be %v, but was %v", "eventstream", event.Name)
	}
	expected := 2
	if int(event.RequestCount) != expected {
		t.Errorf("expected to RequestCount to be %v, but was %v", expected, event.RequestCount)
	}
}

func TestEventStreamErrorPercent(t *testing.T) {
	server := startTestServer()
	defer server.stopTestServer()

	sleepingCommand(t, "errorpercent", 1*time.Millisecond)
	failingCommand(t, "errorpercent", 1*time.Millisecond)
	failingCommand(t, "errorpercent", 1*time.Millisecond)
	failingCommand(t, "errorpercent", 1*time.Millisecond)

	time.Sleep(1 * time.Second)

	event := grabFirstFromStream(t, server.URL)
	if event.ErrorPct != 75 {
		t.Errorf("expected error percent to be %v, found %v", 75, event.ErrorPct)
	}
}
