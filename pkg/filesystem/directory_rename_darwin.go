package filesystem

import (
	"errors"

	"golang.org/x/sys/unix"

	"github.com/mutagen-io/mutagen/pkg/filesystem/internal/syscall"
)

// renameatNoReplaceRetryingOnEINTR is a wrapper around platform-specific
// renameat variants that can perform a renameat operation that fails (with
// EEXIST) if the target already exists. It returns ENOTSUP if the functionality
// is not supported on the target filesystem and ENOSYS if the functionality is
// not supported on the platform as a whole. It retries on EINTR errors and
// returns on the first successful call or non-EINTR error.
func renameatNoReplaceRetryingOnEINTR(oldDirectory int, oldPath string, newDirectory int, newPath string) error {
	for {
		err := syscall.Renameatx_np(oldDirectory, oldPath, newDirectory, newPath, syscall.RENAME_EXCL)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		return err
	}
}
