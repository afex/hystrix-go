package hystrix

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
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
	Convey("given a running event stream", t, func() {
		server := startTestServer()
		defer server.stopTestServer()

		Convey("after 2 successful commands", func() {
			sleepingCommand(t, "eventstream", 1*time.Millisecond)
			sleepingCommand(t, "eventstream", 1*time.Millisecond)

			Convey("the metrics should match", func() {
				event := grabFirstFromStream(t, server.URL)

				So(event.Name, ShouldEqual, "eventstream")
				So(int(event.RequestCount), ShouldEqual, 2)
			})
		})

		Convey("after 1 successful command and 3 unsuccessful commands", func() {
			sleepingCommand(t, "errorpercent", 1*time.Millisecond)
			failingCommand(t, "errorpercent", 1*time.Millisecond)
			failingCommand(t, "errorpercent", 1*time.Millisecond)
			failingCommand(t, "errorpercent", 1*time.Millisecond)

			Convey("the error precentage should be 75", func() {
				event := grabFirstFromStream(t, server.URL)

				So(event.ErrorPct, ShouldEqual, 75)
			})
		})
	})
}
