// +build linux

package filesystem

import (
	"github.com/havoc-io/mutagen/pkg/filesystem/notify"
)

func watchEventMask() notify.Event {
	return notify.InModify | notify.InAttrib |
		notify.InCloseWrite |
		notify.InMovedFrom | notify.InMovedTo |
		notify.InCreate | notify.InDelete |
		notify.InDeleteSelf | notify.InMoveSelf
}
