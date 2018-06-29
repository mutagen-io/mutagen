// +build darwin,cgo

package filesystem

import (
	"os"
	"syscall"

	"github.com/pkg/errors"

	"github.com/rjeczalik/notify"
)

const (
	// recursiveWatchFlags are the flags to use for recursive file watches. When
	// using FSEvents, the FSEventsIsFile flag is necessary to pick up file
	// permission changes, in particular executability.
	recursiveWatchFlags = notify.All | notify.FSEventsIsFile
)

// watchRootParameters specifies the parameters that should be monitored for
// changes when determining whether or not a watch needs to be re-established.
type watchRootParameters struct {
	// deviceID is the device ID.
	deviceID int32
	// inode is the watch root inode.
	inode uint64
}

// probeWatchRoot probes the parameters of the watch root.
func probeWatchRoot(root string) (watchRootParameters, error) {
	if info, err := os.Lstat(root); err != nil {
		if os.IsNotExist(err) {
			return watchRootParameters{}, err
		}
		return watchRootParameters{}, errors.Wrap(err, "unable to grab root metadata")
	} else if stat, ok := info.Sys().(*syscall.Stat_t); !ok {
		return watchRootParameters{}, errors.New("unable to extract raw root metadata")
	} else {
		return watchRootParameters{
			deviceID: stat.Dev,
			inode:    stat.Ino,
		}, nil
	}
}

// watchRootParametersEqual determines whether or not two watch root parameters
// are equal.
func watchRootParametersEqual(first, second watchRootParameters) bool {
	return first.deviceID == second.deviceID &&
		first.inode == second.inode
}
