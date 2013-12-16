hystrix-go
==========

[![Build Status](https://travis-ci.org/afex/hystrix-go.png?branch=master)](https://travis-ci.org/afex/hystrix-go)

```go
import "github.com/afex/hystrix-go/hystrix"

// function we want to protect with a bulkhead
func run(results chan hystrix.Result) {
  // example: access an external service which may be slow or unavailable
  response, err := http.Get("http://service/")
  if err != nil {
    results <- hystrix.Result{ Error: err }
  } else {
    results <- hystrix.Result{ Result: response }
  }
}

// function executed when the "run" function cannot be completed
func fallback(results chan hystrix.Result) {
  // example: when primary service is unavailable, read from a cache instead
  cached_result := ???
  results <- hystrix.Result{ Result: cached_result }
}

command := hystrix.NewCommand(run, fallback)

// run the command synchronously
result := command.Execute()

// or, run the command asynchronously
future := command.Queue()
result = future.Value() // grab result later

// or, define an observer
command.Observer = func(result Result) {
  // act on result parameter
}
command.Observe()
```