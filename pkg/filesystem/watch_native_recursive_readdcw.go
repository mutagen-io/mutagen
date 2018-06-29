// +build windows

package filesystem

import (
	"os"
	"syscall"

	"github.com/pkg/errors"

	"github.com/rjeczalik/notify"
)

const (
	// recursiveWatchFlags are the flags to use for recursive file watches.
	recursiveWatchFlags = notify.All
)

// watchRootParameters specifies the parameters that should be monitored for
// changes when determining whether or not a watch needs to be re-established.
type watchRootParameters struct {
	// fileAttributes are the Windows file attributes
	fileAttributes uint32
	// creationTime is the creation time.
	creationTime syscall.Filetime
}

// probeWatchRoot probes the parameters of the watch root.
func probeWatchRoot(root string) (watchRootParameters, error) {
	if info, err := os.Lstat(root); err != nil {
		if os.IsNotExist(err) {
			return watchRootParameters{}, err
		}
		return watchRootParameters{}, errors.Wrap(err, "unable to grab root metadata")
	} else if stat, ok := info.Sys().(*syscall.Win32FileAttributeData); !ok {
		return watchRootParameters{}, errors.New("unable to extract raw root metadata")
	} else {
		return watchRootParameters{
			fileAttributes: stat.FileAttributes,
			creationTime:   stat.CreationTime,
		}, nil
	}
}

// watchRootParametersEqual determines whether or not two watch root parameters
// are equal.
func watchRootParametersEqual(first, second watchRootParameters) bool {
	return first.fileAttributes == second.fileAttributes &&
		first.creationTime == second.creationTime
}
