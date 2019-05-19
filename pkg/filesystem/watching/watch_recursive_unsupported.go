// +build !darwin,!windows darwin,!cgo

package watching

import (
	"context"
)

const (
	// RecursiveWatchingSupported indicates whether or not the current platform
	// supports native recursive watching.
	RecursiveWatchingSupported = false
)

// WatchRecursive performs recursive watching on platforms which support doing
// so natively. This function is not implemented on this platform and will panic
// if called.
func WatchRecursive(_ context.Context, _ string, _ chan string) error {
	panic("recursive watching not supported on this platform")
}
