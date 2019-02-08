package filesystem

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	// _AT_REMOVEDIR is the numeric representation of the AT_REMOVEDIR flag used
	// with unlinkat.
	_AT_REMOVEDIR = 0x2
)

// syscall6 is a handle for the DragonFly BSD system call implementation in the
// syscall package (which is itself just a thin wrapper around the actual
// implementation in the runtime package). It is wired up via assembly in
// syscall_asm_dragonfly_amd64.s.
func syscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err syscall.Errno)

// _zero is a zero-value that can be used when a valid pointer is needed to 0
// bytes.
var _zero uintptr

// mkdirat is a Go entry point for the mkdirat system call.
func mkdirat(directory int, path string, mode uint32) error {
	return unix.Mkdirat(directory, path, mode)
}

// symlinkat is a Go entry point for the symlinkat system call.
func symlinkat(target string, directory int, path string) error {
	return unix.Symlinkat(target, directory, path)
}

// readlinkat is a Go entry point for the readlinkat system call.
func readlinkat(directory int, path string, buffer []byte) (int, error) {
	// Extract a raw pointer to the path bytes.
	var pathBytes *byte
	pathBytes, err := unix.BytePtrFromString(path)
	if err != nil {
		return 0, err
	}

	// Extract a raw pointer to the buffer bytes.
	var bytesBuffer unsafe.Pointer
	if len(buffer) > 0 {
		bytesBuffer = unsafe.Pointer(&buffer[0])
	} else {
		bytesBuffer = unsafe.Pointer(&_zero)
	}

	// Perform the system call.
	n, _, errnoErr := syscall6(unix.SYS_READLINKAT, uintptr(directory), uintptr(unsafe.Pointer(pathBytes)), uintptr(bytesBuffer), uintptr(len(buffer)), 0, 0)
	if errnoErr != 0 {
		return 0, errnoErr
	}

	// Success.
	return int(n), nil
}

// openat is a Go entry point for the openat system call.
func openat(directory int, path string, flags int, mode uint32) (int, error) {
	return unix.Openat(directory, path, flags, mode)
}

// lstat is a Go entry point for the lstat system call.
func lstat(path string, metadata *unix.Stat_t) error {
	return unix.Lstat(path, metadata)
}

// fstatat is a Go entry point for the fstatat system call.
func fstatat(directory int, path string, metadata *unix.Stat_t, flags int) error {
	return unix.Fstatat(directory, path, metadata, flags)
}

// fchmodat is a Go entry point for the fchmodat system call.
func fchmodat(directory int, path string, mode uint32, flags int) error {
	return unix.Fchmodat(directory, path, mode, flags)
}

// fchownat is a Go entry point for the fchownat system call.
func fchownat(directory int, path string, userId, groupId, flags int) error {
	return unix.Fchownat(directory, path, userId, groupId, flags)
}

// renameat is a Go entry point for the renameat system call.
func renameat(sourceDirectory int, sourcePath string, targetDirectory int, targetPath string) error {
	return unix.Renameat(sourceDirectory, sourcePath, targetDirectory, targetPath)
}

// unlinkat is a Go entry point for the unlinkat system call.
func unlinkat(directory int, path string, flags int) error {
	return unix.Unlinkat(directory, path, flags)
}
