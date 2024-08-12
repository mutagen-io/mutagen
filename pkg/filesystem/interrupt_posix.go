//go:build !windows

package filesystem

import (
	"errors"
	"io"

	"golang.org/x/sys/unix"

	"github.com/mutagen-io/mutagen/pkg/filesystem/internal/syscall"
)

// openatRetryingOnEINTR is a wrapper around the openat system call that retries
// on EINTR errors and returns on the first successful call or non-EINTR error.
func openatRetryingOnEINTR(directory int, path string, flags int, mode uint32) (int, error) {
	for {
		result, err := unix.Openat(directory, path, flags, mode)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		return result, err
	}
}

// readRetryingOnEINTR is a wrapper around the read system call that retries on
// EINTR errors and returns on the first successful call or non-EINTR error.
func readRetryingOnEINTR(file int, buffer []byte) (int, error) {
	for {
		result, err := unix.Read(file, buffer)
		if errors.Is(err, unix.EINTR) {
			continue
		} else if err == nil && result == 0 {
			return 0, io.EOF
		}
		return result, err
	}
}

// seekConsideringEINTR is a direct passthrough to the lseek system call that
// doesn't retry on EINTR. It's only defined to highlight the intentional
// absence of seekRetryingOnEINTR. seekRetryingOnEINTR is left unimplemented
// because it would have to handle cases of partially successful seeks (which
// would be complicated in the case of SEEK_CUR or other relative whence values)
// and because POSIX doesn't specify that lseek can return EINTR. The Go
// standard library and runtime also invoke lseek without retrying on EINTR.
func seekConsideringEINTR(file int, offset int64, whence int) (int64, error) {
	return unix.Seek(file, offset, whence)
}

// closeConsideringEINTR is a direct passthrough to the close system call that
// doesn't retry on EINTR. It's only defined to highlight the intentional
// absence of closeRetryingOnEINTR. closeRetryingOnEINTR is left unimplemented
// because POSIX makes no guarantees about the state of a file descriptor in the
// event of an EINTR error, and thus retrying closure could lead to a race
// condition with file descriptor re-use if the file is, in fact, closed. This
// is the same policy adopted by the Go standard library and runtime.
func closeConsideringEINTR(file int) error {
	return unix.Close(file)
}

// mkdiratRetryingOnEINTR is a wrapper around the mkdirat system call that
// retries on EINTR errors and returns on the first successful call or non-EINTR
// error.
func mkdiratRetryingOnEINTR(directory int, path string, mode uint32) error {
	for {
		err := unix.Mkdirat(directory, path, mode)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		return err
	}
}

// renameatRetryingOnEINTR is a wrapper around the renameat system call that
// retries on EINTR errors and returns on the first successful call or non-EINTR
// error.
func renameatRetryingOnEINTR(oldDirectory int, oldPath string, newDirectory int, newPath string) error {
	for {
		err := unix.Renameat(oldDirectory, oldPath, newDirectory, newPath)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		return err
	}
}

// unlinkatRetryingOnEINTR is a wrapper around the unlinkat system call that
// retries on EINTR errors and returns on the first successful call or non-EINTR
// error.
func unlinkatRetryingOnEINTR(directory int, path string, flags int) error {
	for {
		err := unix.Unlinkat(directory, path, flags)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		return err
	}
}

// fstatRetryingOnEINTR is a wrapper around the fstat system call that retries
// on EINTR errors and returns on the first successful call or non-EINTR error.
func fstatRetryingOnEINTR(file int, metadata *unix.Stat_t) error {
	for {
		err := unix.Fstat(file, metadata)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		return err
	}
}

// fchmodRetryingOnEINTR is a wrapper around the fchmod system call that retries
// on EINTR errors and returns on the first successful call or non-EINTR error.
func fchmodRetryingOnEINTR(file int, mode uint32) error {
	for {
		err := unix.Fchmod(file, mode)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		return err
	}
}

// fstatatRetryingOnEINTR is a wrapper around the fstatat system call that
// retries on EINTR errors and returns on the first successful call or non-EINTR
// error.
func fstatatRetryingOnEINTR(directory int, path string, metadata *unix.Stat_t, flags int) error {
	for {
		err := unix.Fstatat(directory, path, metadata, flags)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		return err
	}
}

// fchmodatRetryingOnEINTR is a wrapper around the fchmodat system call that
// retries on EINTR errors and returns on the first successful call or non-EINTR
// error.
func fchmodatRetryingOnEINTR(directory int, path string, mode uint32, flags int) error {
	for {
		err := unix.Fchmodat(directory, path, mode, flags)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		return err
	}
}

// fchownatRetryingOnEINTR is a wrapper around the fchownat system call that
// retries on EINTR errors and returns on the first successful call or non-EINTR
// error.
func fchownatRetryingOnEINTR(directory int, path string, uid int, gid int, flags int) error {
	for {
		err := unix.Fchownat(directory, path, uid, gid, flags)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		return err
	}
}

// symlinkatRetryingOnEINTR is a wrapper around the symlinkat system call that
// retries on EINTR errors and returns on the first successful call or non-EINTR
// error.
func symlinkatRetryingOnEINTR(target string, directory int, path string) error {
	for {
		err := syscall.Symlinkat(target, directory, path)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		return err
	}
}

// readlinkatRetryingOnEINTR is a wrapper around the readlinkat system call that
// retries on EINTR errors and returns on the first successful call or non-EINTR
// error.
func readlinkatRetryingOnEINTR(directory int, path string, buffer []byte) (int, error) {
	for {
		result, err := syscall.Readlinkat(directory, path, buffer)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		return result, err
	}
}
