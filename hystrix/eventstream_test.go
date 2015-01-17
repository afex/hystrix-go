package hystrix

import (
	"encoding/json"
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
	event := grabFirstFromStream(t, server.URL)

	if event.Name != "eventstream" {
		t.Fatal("metrics did not match command name")
	}
	if event.RequestCount != 1 {
		t.Fatal("metrics did not match request count")
	}
}
