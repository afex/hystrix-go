package hystrix

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

type eventStreamTestServer struct {
	*httptest.Server
	*StreamHandler
}

func (s *eventStreamTestServer) stopTestServer() error {
	s.Close()
	s.Stop()
	Flush()

	return nil
}

func startTestServer() *eventStreamTestServer {
	hystrixStreamHandler := NewStreamHandler()
	hystrixStreamHandler.Start()
	return &eventStreamTestServer{
		httptest.NewServer(hystrixStreamHandler),
		hystrixStreamHandler,
	}
}

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

// grabFirstFromStream reads on the http request until we see the first
// full result printed
func grabFirstCommandFromStream(t *testing.T, url string) streamCmdMetric {
	var event streamCmdMetric

	metrics, done := streamMetrics(t, url)
	for m := range metrics {
		if strings.Contains(m, "HystrixCommand") {
			done <- true
			close(done)

			err := json.Unmarshal([]byte(m), &event)
			if err != nil {
				t.Fatal(err)
			}

			break
		}
	}

	return event
}

func grabFirstThreadPoolFromStream(t *testing.T, url string) streamThreadPoolMetric {
	var event streamThreadPoolMetric

	metrics, done := streamMetrics(t, url)
	for m := range metrics {
		if strings.Contains(m, "HystrixThreadPool") {
			done <- true
			close(done)

			err := json.Unmarshal([]byte(m), &event)
			if err != nil {
				t.Fatal(err)
			}
			break
		}
	}

	return event
}

func streamMetrics(t *testing.T, url string) (chan string, chan bool) {
	metrics := make(chan string, 1)
	done := make(chan bool, 1)

	go func() {
		res, err := http.Get(url)
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
				data = strings.Replace(data, "data:{", "{", 1)
				metrics <- data
				data = ""
			}

			select {
			case _ = <-done:
				close(metrics)
				return
			default:
			}
		}
	}()

	return metrics, done
}

func TestEventStream(t *testing.T) {
	Convey("given a running event stream", t, func() {
		server := startTestServer()
		defer server.stopTestServer()

		Convey("after 2 successful commands", func() {
			sleepingCommand(t, "eventstream", 1*time.Millisecond)
			sleepingCommand(t, "eventstream", 1*time.Millisecond)

			Convey("request count should be 2", func() {
				event := grabFirstCommandFromStream(t, server.URL)

				So(event.Name, ShouldEqual, "eventstream")
				So(int(event.RequestCount), ShouldEqual, 2)
			})
		})

		Convey("after 1 successful command and 2 unsuccessful commands", func() {
			sleepingCommand(t, "errorpercent", 1*time.Millisecond)
			failingCommand(t, "errorpercent", 1*time.Millisecond)
			failingCommand(t, "errorpercent", 1*time.Millisecond)

			Convey("the error precentage should be 67", func() {
				metric := grabFirstCommandFromStream(t, server.URL)

				So(metric.ErrorPct, ShouldEqual, 67)
			})
		})
	})
}

func TestClientCancelEventStream(t *testing.T) {
	Convey("given a running event stream", t, func() {
		server := startTestServer()
		defer server.stopTestServer()

		sleepingCommand(t, "eventstream", 1*time.Millisecond)

		Convey("after a client connects", func() {
			req, err := http.NewRequest("GET", server.URL, nil)
			if err != nil {
				t.Fatal(err)
			}
			// use a transport so we can cancel the stream when we're done - in 1.5 this is much easier
			tr := &http.Transport{}
			client := &http.Client{Transport: tr}
			wait := make(chan struct{})
			afterFirstRead := &sync.WaitGroup{}
			afterFirstRead.Add(1)

			go func() {
				afr := afterFirstRead
				buf := []byte{0}
				res, err := client.Do(req)
				if err != nil {
					t.Fatal(err)
				}
				defer res.Body.Close()

				for {
					select {
					case <-wait:
						//wait for master goroutine to break us out
						tr.CancelRequest(req)
						return
					default:
						//read something
						_, err = res.Body.Read(buf)
						if err != nil {
							t.Fatal(err)
						}
						if afr != nil {
							afr.Done()
							afr = nil
						}
					}
				}
			}()
			// need to make sure our request has round-tripped to the server
			afterFirstRead.Wait()

			Convey("it should be registered", func() {
				server.StreamHandler.mu.RLock()
				So(len(server.StreamHandler.requests), ShouldEqual, 1)
				server.StreamHandler.mu.RUnlock()
				Convey("after client disconnects", func() {
					// let the request be cancelled and the body closed
					close(wait)
					// wait for the server to clean up
					time.Sleep(2000 * time.Millisecond)
					Convey("it should be detected as disconnected and de-registered", func() {
						//confirm we have 0 clients
						server.StreamHandler.mu.RLock()
						So(len(server.StreamHandler.requests), ShouldEqual, 0)
						server.StreamHandler.mu.RUnlock()
					})
				})
			})
		})
	})
}

func TestThreadPoolStream(t *testing.T) {
	Convey("given a running event stream", t, func() {
		server := startTestServer()
		defer server.stopTestServer()

		Convey("after a successful command", func() {
			sleepingCommand(t, "threadpool", 1*time.Millisecond)
			metric := grabFirstThreadPoolFromStream(t, server.URL)

			Convey("the rolling count of executions should increment", func() {
				So(metric.RollingCountThreadsExecuted, ShouldEqual, 1)
			})

			Convey("the pool size should be 10", func() {
				So(metric.CurrentPoolSize, ShouldEqual, 10)
			})
		})
	})
}
