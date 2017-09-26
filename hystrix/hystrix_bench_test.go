package hystrix

import (
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"testing"
	"time"
)

func BenchmarkGo(b *testing.B) {
	ConfigureCommand("bench-command", CommandConfig{
		MaxConcurrentRequests:       1000,
		QueueSizeRejectionThreshold: 50,
		Timeout:                     100,
	})
	initialGoRoutine := runtime.NumGoroutine()
	wg := sync.WaitGroup{}
	wg.Add(10000)
	for i := 0; i < 10000; i++ {
		go func() {
			_ = Do("bench-command", func() error {
				time.Sleep(5 * time.Millisecond)
				return nil
			}, nil)
			wg.Done()
		}()
	}
	wg.Wait()
	time.Sleep(1 * time.Second)
	finalGoRoutine := runtime.NumGoroutine()
	// allow an approximate 10 goroutines to exist, they will soon be garbage collected
	if finalGoRoutine-initialGoRoutine > 10 {
		b.Fail()
	}
	_ = pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
}
