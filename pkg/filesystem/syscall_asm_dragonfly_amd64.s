// Re-exposure of DragonFly BSD system call implementation from the syscall
// package. Based on (but modified from):
// https://github.com/golang/sys/blob/302c3dd5f1cc82baae8e44d9c3178e89b6e2b345/unix/asm_dragonfly_amd64.s.
//
// The original code license:
//
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !gccgo

#include "textflag.h"

//
// System call support for AMD64, DragonFly
//

// Just jump to package syscall's implementation for all these functions.
// The runtime may know about them.

TEXT    ·syscall6(SB),NOSPLIT,$0-80
    JMP syscall·Syscall6(SB)
