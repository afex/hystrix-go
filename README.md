hystrix-go
==========

[![Build Status](https://travis-ci.org/afex/hystrix-go.png?branch=master)](https://travis-ci.org/afex/hystrix-go)

[Hystrix](https://github.com/Netflix/Hystrix) is a great project from Netflix. Check it out.

> Hystrix is a latency and fault tolerance library designed to isolate points of access to remote systems, services and 3rd party libraries, stop cascading failure and enable resilience in complex distributed systems where failure is inevitable.

I think the Hystrix patterns of programmer-defined fallbacks and adaptive health monitoring are good for any distributed system. Go routines and channels are great concurrency primitives, but don't directly help our application stay available during failures.

hystrix-go aims to allow Go programmers to easily build applications with similar execution semantics of the Java-based Hystrix library.

For more about how Hystrix works, refer to the [Java Hystrix wiki](https://github.com/Netflix/Hystrix/wiki)

Example Code
------------

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