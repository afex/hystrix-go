hystrix-go
==========

[![Build Status](https://travis-ci.org/afex/hystrix-go.png?branch=master)](https://travis-ci.org/afex/hystrix-go)

```go
import "github.com/afex/hystrix-go/hystrix"

// function we want to protect with a bulkhead
func run(results chan hystrix.Result) {
  // i.e. access an external service which may be abnormally slow or unavailable
  response, err := http.Get("http://service/")
  if err != nil {
    results <- hystrix.Result{ Error: err }
    return
  }
  
  results <- hystrix.Result{ Result: string(body) }
}

// function executed when the "run" function cannot be completed
func fallback(results chan hystrix.Result) {
  cached_result := // when primary service is unavailable, read from the cache instead
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