// +build !windows,!linux,!darwin darwin,!cgo

package filesystem

import (
	"context"

	"github.com/pkg/errors"
)

func watchNative(_ context.Context, _ string, _ chan struct{}, _ uint32) error {
	return errors.New("native watching not supported on this platform")
}
