// +build !windows,!plan9

// TODO: Figure out what to do for Plan 9. It doesn't have syscall.Stat_t.

package filesystem

import (
	"os"
	"syscall"

	"github.com/pkg/errors"
)

func GetOwnership(info os.FileInfo) (int, int, error) {
	if stat, ok := info.Sys().(*syscall.Stat_t); !ok {
		return 0, 0, errors.New("unable to extract raw stat information")
	} else {
		return int(stat.Uid), int(stat.Gid), nil
	}
}

func SetOwnership(path string, uid, gid int) error {
	return os.Lchown(path, uid, gid)
}
