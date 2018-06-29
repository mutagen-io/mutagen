// +build !windows,!darwin darwin,!cgo

package filesystem

import (
	"context"

	"github.com/pkg/errors"
)

func watchNative(_ context.Context, _ string, _ chan struct{}) error {
	return errors.New("native recursive watching not supported")
}
