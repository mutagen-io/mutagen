package syscall

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	// AT_REMOVEDIR is the numeric representation of the AT_REMOVEDIR flag used
	// with the unlinkat system call.
	AT_REMOVEDIR = 0x2
)

// syscall6 is a handle for the DragonFly BSD system call implementation in the
// syscall package (which is itself just a thin wrapper around the actual
// implementation in the runtime package). It is wired up via assembly in
// syscall_asm_dragonfly_amd64.s.
func syscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err syscall.Errno)

// _zero is a zero-value that can be used when a valid pointer is needed to 0
// bytes.
var _zero uintptr

// Symlinkat is a Go entry point for the symlinkat system call.
func Symlinkat(target string, directory int, path string) error {
	return unix.Symlinkat(target, directory, path)
}

// Readlinkat is a Go entry point for the readlinkat system call.
func Readlinkat(directory int, path string, buffer []byte) (int, error) {
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
