package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/afex/hystrix-go/hystrix/metric_collector"
	"github.com/afex/hystrix-go/plugins"
	"github.com/cactus/go-statsd-client/statsd"
)

func main() {
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

	http.HandleFunc("/", handle)
	log.Fatal(http.ListenAndServe(":8888", nil))
}

func handle(w http.ResponseWriter, r *http.Request) {
	hystrix.Go("test", func() error {
		time.Sleep(60 * time.Millisecond)
		return nil
	}, func(err error) error {
		return nil
	})
}
