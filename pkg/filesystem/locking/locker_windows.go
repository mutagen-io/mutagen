// Windows file locking implementation based on (but heavily modified from)
// https://github.com/golang/build/blob/4821e1d4e1dd5d386f53f1e869ced293dd18f44a/cmd/builder/filemutex_windows.go.
//
// The original code license:
//
// Copyright (c) 2009 The Go Authors. All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//    * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//    * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//    * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
//
// The original license header inside the code itself:
//
// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package locking

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32     = windows.NewLazySystemDLL("kernel32.dll")
	lockFileEx   = kernel32.NewProc("LockFileEx")
	unlockFileEx = kernel32.NewProc("UnlockFileEx")
)

const (
	LOCKFILE_EXCLUSIVE_LOCK   = 2
	LOCKFILE_FAIL_IMMEDIATELY = 1
)

func callLockFileEx(
	handle syscall.Handle,
	flags,
	reserved,
	lockLow,
	lockHigh uint32,
	overlapped *syscall.Overlapped,
) (err error) {
	r1, _, e1 := syscall.Syscall6(
		lockFileEx.Addr(),
		6,
		uintptr(handle),
		uintptr(flags),
		uintptr(reserved),
		uintptr(lockLow),
		uintptr(lockHigh),
		uintptr(unsafe.Pointer(overlapped)),
	)
	if r1 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func callunlockFileEx(
	handle syscall.Handle,
	reserved,
	lockLow,
	lockHigh uint32,
	overlapped *syscall.Overlapped,
) (err error) {
	r1, _, e1 := syscall.Syscall6(
		unlockFileEx.Addr(),
		5,
		uintptr(handle),
		uintptr(reserved),
		uintptr(lockLow),
		uintptr(lockHigh),
		uintptr(unsafe.Pointer(overlapped)),
		0,
	)
	if r1 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

// Lock attempts to acquire the file lock.
func (l *Locker) Lock(block bool) error {
	var ol syscall.Overlapped
	flags := uint32(LOCKFILE_EXCLUSIVE_LOCK)
	if !block {
		flags |= LOCKFILE_FAIL_IMMEDIATELY
	}
	return callLockFileEx(syscall.Handle(l.file.Fd()), flags, 0, 1, 0, &ol)
}

// Unlock releases the file lock.
func (l *Locker) Unlock() error {
	var ol syscall.Overlapped
	return callunlockFileEx(syscall.Handle(l.file.Fd()), 0, 1, 0, &ol)
}
