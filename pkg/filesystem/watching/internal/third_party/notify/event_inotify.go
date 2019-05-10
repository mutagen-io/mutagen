// Subset of https://github.com/rjeczalik/notify extracted and modified to
// expose watcher functionality directly. Originally extracted from the
// following revision:
// https://github.com/rjeczalik/notify/tree/52ae50d8490436622a8941bd70c3dbe0acdd4bbf
//
// The original code license:
//
// The MIT License (MIT)
//
// Copyright (c) 2014-2015 The Notify Authors
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
//
// The original license header inside the code itself:
//
// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

// +build linux

package notify

import "golang.org/x/sys/unix"

// Platform independent event values.
const (
	osSpecificCreate Event = 0x100000 << iota
	osSpecificRemove
	osSpecificWrite
	osSpecificRename
	// internal
	// recursive is used to distinguish recursive eventsets from non-recursive ones
	recursive
	// omit is used for dispatching internal events; only those events are sent
	// for which both the event and the watchpoint has omit in theirs event sets.
	omit
)

// Inotify specific masks are legal, implemented events that are guaranteed to
// work with notify package on linux-based systems.
const (
	InAccess       = Event(unix.IN_ACCESS)        // File was accessed
	InModify       = Event(unix.IN_MODIFY)        // File was modified
	InAttrib       = Event(unix.IN_ATTRIB)        // Metadata changed
	InCloseWrite   = Event(unix.IN_CLOSE_WRITE)   // Writtable file was closed
	InCloseNowrite = Event(unix.IN_CLOSE_NOWRITE) // Unwrittable file closed
	InOpen         = Event(unix.IN_OPEN)          // File was opened
	InMovedFrom    = Event(unix.IN_MOVED_FROM)    // File was moved from X
	InMovedTo      = Event(unix.IN_MOVED_TO)      // File was moved to Y
	InCreate       = Event(unix.IN_CREATE)        // Subfile was created
	InDelete       = Event(unix.IN_DELETE)        // Subfile was deleted
	InDeleteSelf   = Event(unix.IN_DELETE_SELF)   // Self was deleted
	InMoveSelf     = Event(unix.IN_MOVE_SELF)     // Self was moved
)

var osestr = map[Event]string{
	InAccess:       "notify.InAccess",
	InModify:       "notify.InModify",
	InAttrib:       "notify.InAttrib",
	InCloseWrite:   "notify.InCloseWrite",
	InCloseNowrite: "notify.InCloseNowrite",
	InOpen:         "notify.InOpen",
	InMovedFrom:    "notify.InMovedFrom",
	InMovedTo:      "notify.InMovedTo",
	InCreate:       "notify.InCreate",
	InDelete:       "notify.InDelete",
	InDeleteSelf:   "notify.InDeleteSelf",
	InMoveSelf:     "notify.InMoveSelf",
}

// Inotify behavior events are not **currently** supported by notify package.
const (
	inDontFollow = Event(unix.IN_DONT_FOLLOW)
	inExclUnlink = Event(unix.IN_EXCL_UNLINK)
	inMaskAdd    = Event(unix.IN_MASK_ADD)
	inOneshot    = Event(unix.IN_ONESHOT)
	inOnlydir    = Event(unix.IN_ONLYDIR)
)

type event struct {
	sys   unix.InotifyEvent
	path  string
	event Event
}

func (e *event) Event() Event         { return e.event }
func (e *event) Path() string         { return e.path }
func (e *event) Sys() interface{}     { return &e.sys }
func (e *event) isDir() (bool, error) { return e.sys.Mask&unix.IN_ISDIR != 0, nil }
