// +build windows

package filesystem

import (
	"github.com/rjeczalik/notify"
)

const (
	// recursiveWatchFlags are the flags to use for recursive file watches.
	recursiveWatchFlags = notify.All
)
