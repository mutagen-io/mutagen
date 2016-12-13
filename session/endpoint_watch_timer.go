// +build ignore

// +build !windows,!darwin darwin,!cgo

package session

import (
	"time"

	"github.com/pkg/errors"

	"golang.org/x/net/context"
)

const (
	watchTickInterval = 5 * time.Second
)

func watch(ctx context.Context, _ string, events chan struct{}) error {
	// Create a ticker. Ensure it is stopped when this method exits.
	ticker := time.NewTicker(watchTickInterval)
	defer ticker.Stop()

	// Poll for the next notification or cancellation.
	for {
		select {
		case <-ticker.C:
			// Forward a tick event in a non-blocking manner.
			select {
			case events <- struct{}{}:
			default:
			}
		case <-ctx.Done():
			// Abort in the event of cancellation.
			return errors.New("watch cancelled")
		}
	}
}
