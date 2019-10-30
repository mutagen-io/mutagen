package contextutil

import (
	"context"
)

// IsCancelled returns whether or not the context's Done channel is closed.
func IsCancelled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
