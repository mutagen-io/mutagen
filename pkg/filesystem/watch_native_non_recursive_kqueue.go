// +build darwin,!cgo dragonfly freebsd netbsd openbsd

package filesystem

import (
	"github.com/havoc-io/mutagen/pkg/filesystem/notify"
)

func watchEventMask() notify.Event {
	return notify.NoteDelete | notify.NoteWrite | notify.NoteExtend |
		notify.NoteAttrib | notify.NoteRename
}
