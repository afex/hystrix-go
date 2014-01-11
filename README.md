hystrix-go
==========

[![Build Status](https://travis-ci.org/afex/hystrix-go.png?branch=master)](https://travis-ci.org/afex/hystrix-go)

[Hystrix](https://github.com/Netflix/Hystrix) is a great project from Netflix. Check it out.

> Hystrix is a latency and fault tolerance library designed to isolate points of access to remote systems, services and 3rd party libraries, stop cascading failure and enable resilience in complex distributed systems where failure is inevitable.

I think the Hystrix patterns of programmer-defined fallbacks and adaptive health monitoring are good for any distributed system. Go routines and channels are great concurrency primitives, but don't directly help our application stay available during failures.

hystrix-go aims to allow Go programmers to easily build applications with similar execution semantics of the Java-based Hystrix library.

For more about how Hystrix works, refer to the [Java Hystrix wiki](https://github.com/Netflix/Hystrix/wiki)


How to use
----------

__hystrix-go is a learning project for me, and as such I expect the API to transform as I learn more appropriate Go patterns.__

```go
import "github.com/afex/hystrix-go/hystrix"
```

### Define run and fallback functions

First, we'll need to define your application logic which relies on external systems. This is the "run" function and when the system is healthy will be the only thing which executes.

```go
func run(results chan hystrix.Result) {
  // example: access an external service which may be slow or unavailable
  response, err := http.Get("http://service/")
  if err != nil {
    results <- hystrix.Result{ Error: err }
  } else {
    results <- hystrix.Result{ Result: response }
  }
}
```

Next, we define the "fallback" function.  This is triggered whenever the run function is unable to complete, based on a [variety of health checks](https://github.com/Netflix/Hystrix/wiki/How-it-Works).

```go
func fallback(results chan hystrix.Result) {
  // example: when primary service is unavailable, read from a cache instead
  cached_result := ???
  results <- hystrix.Result{ Result: cached_result }
}
```

### Synchronous execution

Start a command, and wait for it to finish.

```go
command := hystrix.NewCommand(run, fallback)

result := command.Execute()
```

### Asynchronous execution

Start a command, and receive a channel to grab the response later.

```go
command := hystrix.NewCommand(run, fallback)

channel := command.Queue()

result = <-channel
```