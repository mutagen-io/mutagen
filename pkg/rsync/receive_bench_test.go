package rsync

import (
	"context"
	"sync"
	"testing"
)

func BenchmarkPreemptionCheckOverhead(b *testing.B) {
	// Grab the background context.
	ctx := context.Background()

	// Reset the benchmark timer to exclude the setup time.
	b.ResetTimer()

	// Perform the benchmark.
	for i := 0; i < b.N; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func newBenchmarkCallback() func() {
	// Create a mutex.
	mutex := &sync.Mutex{}

	// Create a callback.
	return func() {
		mutex.Lock()
		mutex.Unlock()
	}
}

func BenchmarkMonitoringCallbackOverhead(b *testing.B) {
	// Create a sample monitoring callback. We dynamically allocate this
	// function to avoid the possibility of inlining and thus more realistically
	// simulate usage.
	callback := newBenchmarkCallback()

	// Reset the benchmark timer to exclude the setup time.
	b.ResetTimer()

	// Perform the benchmark.
	for i := 0; i < b.N; i++ {
		callback()
	}
}
