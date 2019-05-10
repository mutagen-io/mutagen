// Package syscall is an internal POSIX system call compatibility shim needed to
// ensure the availability of certain constants and functions across all
// supported POSIX platforms. It will go away once golang.org/x/sys/unix adds
// these definitions and implementations for all necessary platforms.
package syscall
