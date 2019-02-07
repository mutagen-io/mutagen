// Re-exposure of Solaris system call implementation from the syscall package
// (which is itself just a thin wrapper around the actual system implementation
// in the runtime package). Based on (but modified from):
// https://github.com/golang/sys/blob/302c3dd5f1cc82baae8e44d9c3178e89b6e2b345/unix/asm_solaris_amd64.s.
//
// The original code license:
//
// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !gccgo

#include "textflag.h"

//
// System calls for amd64, Solaris are implemented in runtime/syscall_solaris.go
//

TEXT ·sysvicall6(SB),NOSPLIT,$0-88
    JMP syscall·sysvicall6(SB)
