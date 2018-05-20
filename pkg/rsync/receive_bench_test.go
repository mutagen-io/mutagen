package rsync

import (
	"context"
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
