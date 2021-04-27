package filesystem

import (
	"golang.org/x/sys/unix"

	"github.com/mutagen-io/mutagen/pkg/state"
)

// renameat2FailedWithENOSYS tracks if renameat2 previously failed with ENOSYS.
var renameat2FailedWithENOSYS state.Marker

// renameatNoReplaceRetryingOnEINTR is a wrapper around platform-specific
// renameat variants that can perform a renameat operation that fails (with
// EEXIST) if the target already exists. It returns ENOTSUP if the functionality
// is not supported on the target filesystem and ENOSYS if the functionality is
// not supported on the platform as a whole. It retries on EINTR errors and
// returns on the first successful call or non-EINTR error.
func renameatNoReplaceRetryingOnEINTR(oldDirectory int, oldPath string, newDirectory int, newPath string) error {
	// If renameat2 is known to be unavailable, then return immediately.
	if renameat2FailedWithENOSYS.Marked() {
		return unix.ENOSYS
	}

	// Loop until renameat2 completes with a return value other that EINTR.
	for {
		err := unix.Renameat2(oldDirectory, oldPath, newDirectory, newPath, unix.RENAME_NOREPLACE)
		if err == unix.EINTR {
			continue
		} else if err == unix.EINVAL {
			// HACK: On Linux, using RENAME_NOREPLACE with a target filesystem
			// that doesn't support it will yield EINVAL. To keep consistency
			// with other renameatNoReplaceRetryingOnEINTR implementations, we
			// alias this case to ENOTSUP. This is a little hacky because EINVAL
			// can also be returned in cases where paths are incompatible (due
			// to one being a subdirectory of the other), but those cases will
			// just be found by the fallback method used in the case of ENOTSUP.
			// All other cases of EINVAL don't apply to this very controlled
			// invocation of renameat2. The only other error we'd need to
			// consider would be ENOSYS, but that we don't need to alias.
			return unix.ENOTSUP
		} else if err == unix.ENOSYS {
			renameat2FailedWithENOSYS.Mark()
		}
		return err
	}
}
