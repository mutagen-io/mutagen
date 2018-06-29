// +build darwin,cgo

package filesystem

import (
	"github.com/rjeczalik/notify"
)

const (
	// recursiveWatchFlags are the flags to use for recursive file watches. When
	// using FSEvents, the FSEventsIsFile flag is necessary to pick up file
	// permission changes, in particular executability.
	recursiveWatchFlags = notify.All | notify.FSEventsIsFile
)
