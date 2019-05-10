// Windows path handling tests based on (but modified from)
// https://github.com/golang/go/blob/da0d1a44bac379f5acedb1933f85400de08f4ac6/src/os/path_windows_test.go
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
// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package os

import (
	"strings"
	"testing"
)

func TestFixLongPath(t *testing.T) {
	// 248 is long enough to trigger the longer-than-248 checks in
	// fixLongPath, but short enough not to make a path component
	// longer than 255, which is illegal on Windows. (which
	// doesn't really matter anyway, since this is purely a string
	// function we're testing, and it's not actually being used to
	// do a system call)
	veryLong := "l" + strings.Repeat("o", 248) + "ng"
	for _, test := range []struct{ in, want string }{
		// Short; unchanged:
		{`C:\short.txt`, `C:\short.txt`},
		{`C:\`, `C:\`},
		{`C:`, `C:`},
		// The "long" substring is replaced by a looooooong
		// string which triggers the rewriting. Except in the
		// cases below where it doesn't.
		{`C:\long\foo.txt`, `\\?\C:\long\foo.txt`},
		{`C:/long/foo.txt`, `\\?\C:\long\foo.txt`},
		{`C:\long\foo\\bar\.\baz\\`, `\\?\C:\long\foo\bar\baz`},
		{`\\unc\path`, `\\unc\path`},
		{`long.txt`, `long.txt`},
		{`C:long.txt`, `C:long.txt`},
		{`c:\long\..\bar\baz`, `c:\long\..\bar\baz`},
		{`\\?\c:\long\foo.txt`, `\\?\c:\long\foo.txt`},
		{`\\?\c:\long/foo.txt`, `\\?\c:\long/foo.txt`},
	} {
		in := strings.ReplaceAll(test.in, "long", veryLong)
		want := strings.ReplaceAll(test.want, "long", veryLong)
		if got := FixLongPath(in); got != want {
			got = strings.ReplaceAll(got, veryLong, "long")
			t.Errorf("fixLongPath(%q) = %q; want %q", test.in, got, test.want)
		}
	}
}
