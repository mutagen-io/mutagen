//go:build !windows && !linux && !darwin
// +build !windows,!linux,!darwin

package filesystem

import (
	"golang.org/x/sys/unix"
)

// renameatNoReplaceRetryingOnEINTR is a wrapper around platform-specific
// renameat variants that can perform a renameat operation that fails (with
// EEXIST) if the target already exists. It returns ENOTSUP if the functionality
// is not supported on the target filesystem and ENOSYS if the functionality is
// not supported on the platform as a whole. It retries on EINTR errors and
// returns on the first successful call or non-EINTR error.
func renameatNoReplaceRetryingOnEINTR(_ int, _ string, _ int, _ string) error {
	return unix.ENOSYS
}
