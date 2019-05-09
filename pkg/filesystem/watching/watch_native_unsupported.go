// +build !windows,!linux,!darwin darwin,!cgo

package watching

import (
	"context"

	"github.com/pkg/errors"
)

// watchNative attempts to perform efficient watching using the operating
// system's native filesystem watching facilities.
func watchNative(_ context.Context, _ string, _ chan struct{}, _ uint32) error {
	return errors.New("native watching not supported on this platform")
}
