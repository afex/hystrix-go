// Package main implements an http server which executes a hystrix command each request and
// sends metrics to a statsd instance to aid performance testing.
package main

import (
	"flag"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"time"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/afex/hystrix-go/hystrix/metric_collector"
	"github.com/afex/hystrix-go/plugins"
	"github.com/cactus/go-statsd-client/statsd"
)

const (
	deltaWindow = 10
	minDelay    = 35
	maxDelay    = 55
)

var (
	delay int
)

const (
	up = iota
	down
)

func init() {
	delay = minDelay
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	statsdHost := flag.String("statsd", "", "Statsd host to record load test metrics")
	flag.Parse()

	_, err := statsd.NewClient(*statsdHost, "hystrix.loadtest.driver")
	if err != nil {
		log.Fatalf("could not initialize statsd client: %v", err)
	}

	c, err := plugins.InitializeStatsdCollector(&plugins.StatsdCollectorConfig{
		StatsdAddr: *statsdHost,
		Prefix:     "hystrix.loadtest.circuits",
	})
	if err != nil {
		log.Fatalf("could not initialize statsd client: %v", err)
	}
	metricCollector.Registry.Register(c.NewStatsdCollector)

	hystrix.ConfigureCommand("test", hystrix.CommandConfig{
		Timeout: 50,
	})

	go rotateDelay()

	http.HandleFunc("/", handle)
	log.Print("starting server")
	log.Fatal(http.ListenAndServe(":8888", nil))
}

func handle(w http.ResponseWriter, r *http.Request) {
	done := make(chan struct{}, 1)
	errChan := hystrix.Go("test", func() error {
		delta := rand.Intn(deltaWindow)
		time.Sleep(time.Duration(delay+delta) * time.Millisecond)
		done <- struct{}{}
		return nil
	}, func(err error) error {
		done <- struct{}{}
		return nil
	})

	select {
	case err := <-errChan:
		http.Error(w, err.Error(), 500)
	case <-done:
		w.Write([]byte("OK"))
	}
}

func rotateDelay() {
	direction := up
	for {
		if direction == up && delay == maxDelay {
			direction = down
		}
		if direction == down && delay == minDelay {
			direction = up
		}

		if direction == up {
			delay += 1
		} else {
			delay -= 1
		}

		time.Sleep(10 * time.Second)
		log.Printf("setting delay to %v", delay)
	}
}
