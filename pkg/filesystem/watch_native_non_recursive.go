// +build !windows,!darwin darwin,!cgo

package filesystem

import (
	"context"

	"github.com/pkg/errors"
)

func watchRecursiveHome(_ context.Context, _ string, _ chan struct{}) error {
	return errors.New("native watching not supported")
}
