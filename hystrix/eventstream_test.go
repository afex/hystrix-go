package hystrix

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEventStream(t *testing.T) {
	hystrixStreamHandler := NewStreamHandler()
	hystrixStreamHandler.Start()
	defer hystrixStreamHandler.Stop()
	server := httptest.NewServer(hystrixStreamHandler)
	defer server.Close()

	done := make(chan bool)
	errChan := Go("eventstream", func() error {
		time.Sleep(100 * time.Millisecond)
		done <- true
		return nil
	}, nil)

	select {
	case _ = <-done:
		// do nothing
	case err := <-errChan:
		t.Fatal(err)
	}

	res, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

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

	if event.Name != "eventstream" {
		t.Fatal("metrics did not match command name")
	}
	if event.RequestCount != 1 {
		t.Fatal("metrics did not match request count")
	}
}
