package syscall

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// libcFunction is a handle type for AIX libc functions.
type libcFunction uintptr

// syscall6 is a handle for the AIX system call implementation in the syscall
// package (which is itself just a thin wrapper around the actual implementation
// in the runtime package). It is wired up via assembly in
// syscall_asm_aix_ppc64.s.
func syscall6(trap, nargs, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err syscall.Errno)

//go:cgo_import_dynamic libc_symlinkat symlinkat "libc.a/shr_64.o"
//go:cgo_import_dynamic libc_readlinkat readlinkat "libc.a/shr_64.o"

//go:linkname procSymlinkat libc_symlinkat
//go:linkname procReadlinkat libc_readlinkat

var (
	// procSymlinkat is a handle for the symlinkat libc function.
	procSymlinkat,
	// procReadlinkat is a handle for the readlinkat libc function.
	procReadlinkat libcFunction
)

// Symlinkat is a Go entry point for the symlinkat system call.
func Symlinkat(target string, directory int, path string) error {
	// Extract a raw pointer to the target path bytes.
	var targetBytes *byte
	var err error
	targetBytes, err = unix.BytePtrFromString(target)
	if err != nil {
		return err
	}

	// Extract a raw pointer to the path bytes.
	var pathBytes *byte
	pathBytes, err = unix.BytePtrFromString(path)
	if err != nil {
		return err
	}

	// Perform the system call.
	_, _, errnoErr := syscall6(uintptr(unsafe.Pointer(&procSymlinkat)), 3, uintptr(unsafe.Pointer(targetBytes)), uintptr(directory), uintptr(unsafe.Pointer(pathBytes)), 0, 0, 0)
	if errnoErr != 0 {
		return errnoErr
	}

	// Success.
	return nil
}

// Readlinkat is a Go entry point for the readlinkat system call.
func Readlinkat(directory int, path string, buffer []byte) (int, error) {
	// Extract a raw pointer to the path bytes.
	var pathBytes *byte
	var err error
	pathBytes, err = unix.BytePtrFromString(path)
	if err != nil {
		return 0, err
	}

	// Extract a raw pointer to the buffer bytes.
	var bufferBytes *byte
	if len(buffer) > 0 {
		bufferBytes = &buffer[0]
	}

	// Perform the system call.
	n, _, errnoErr := syscall6(uintptr(unsafe.Pointer(&procReadlinkat)), 4, uintptr(directory), uintptr(unsafe.Pointer(pathBytes)), uintptr(unsafe.Pointer(bufferBytes)), uintptr(len(buffer)), 0, 0)
	if errnoErr != 0 {
		return int(n), errnoErr
	}

	// Success.
	return int(n), nil
}
