package hystrix

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestDoCDoesNotFallbackAfterSuccessWhenContextCanceled(t *testing.T) {
	defer Flush()

	const name = "doc-cancel-after-success"
	ConfigureCommand(name, CommandConfig{
		Timeout:               1000,
		MaxConcurrentRequests: 32,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var fallbacks int64
	testAfterSuccessCommitted = func() {
		cancel()
		time.Sleep(20 * time.Millisecond)
	}
	defer func() { testAfterSuccessCommitted = nil }()

	err := DoC(ctx, name, func(context.Context) error {
		return nil
	}, func(context.Context, error) error {
		atomic.AddInt64(&fallbacks, 1)
		return nil
	})
	if err != nil {
		t.Fatalf("DoC returned %v", err)
	}
	if n := atomic.LoadInt64(&fallbacks); n != 0 {
		t.Fatalf("fallback ran %d times after a successful run", n)
	}
}
