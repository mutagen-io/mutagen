// Filesystem walking implementation that provides an interface compatible with
// Go's standard path/filepath.Walk, but with slightly different semantics and
// significantly enhanced speed. The implementation was based on Go at 1.10.3,
// with the code for that revision available here:
// https://github.com/golang/go/blob/fe8a0d12b14108cbe2408b417afcaab722b0727c/src/path/filepath/path.go
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
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filesystem

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// walkRecursive is the recursive entry point underlying Walk.
func walkRecursive(path string, info os.FileInfo, visitor filepath.WalkFunc) error {
	// If this isn't a directory, then just visit it directly.
	if !info.IsDir() {
		return visitor(path, info, nil)
	}

	// Read directory contents.
	contents, contentErr := DirectoryContentsByPath(path)

	// Visit the directory, passing the visitor any error that occurred in
	// reading contents.
	visitErr := visitor(path, info, contentErr)

	// If we can't traverse into the directory, then we needn't continue.
	if contentErr != nil || visitErr != nil {
		return visitErr
	}

	// Process contents.
	for _, c := range contents {
		if err := walkRecursive(filepath.Join(path, c.Name()), c, visitor); err != nil {
			if err == filepath.SkipDir {
				if !c.IsDir() {
					return errors.New("directory skip requested for non-directory")
				}
			} else {
				return err
			}
		}
	}

	// Success.
	return nil
}

// Walk provides a faster implementation of path/filepath.Walk with slightly
// different semantics. In particular, it only walks directory contents after
// metadata for the directory has been collected and does not sort entries by
// name. It gains speed by avoiding sorting and, more importantly, using
// os.File.Readdir for vastly more efficient traversal on Windows.
func Walk(root string, visitor filepath.WalkFunc) error {
	// Create our error result.
	var result error

	// Grab information on the watch root.
	if info, err := os.Lstat(root); err != nil {
		result = visitor(root, nil, err)
	} else {
		result = walkRecursive(root, info, visitor)
	}

	// If the visitor has requested skipping the root, then everything is okay.
	if result == filepath.SkipDir {
		return nil
	}

	// Done.
	return result
}
