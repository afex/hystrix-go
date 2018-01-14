integration app to measure behavior of circuits under load.

`go run service/main.go -statsd mystatsdhost:8125`

`ab -n 10000000 -c 10 http://localhost:8888/`

To test buffer queue, increase the concurrency of the bench tool

The queue rate is under the metric `hystrix.loadtest.circuits.test.queueLength.count` 

`ab -n 10000000 -c 11 http://localhost:8888/`
