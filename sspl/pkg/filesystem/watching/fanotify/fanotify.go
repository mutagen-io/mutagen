//go:build linux && mutagensspl

// Copyright (c) 2020-present Docker, Inc.
//
// This program is free software: you can redistribute it and/or modify it under
// the terms of the Server Side Public License, version 1, as published by
// MongoDB, Inc.
//
// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
// FOR A PARTICULAR PURPOSE. See the Server Side Public License for more
// details.
//
// You should have received a copy of the Server Side Public License along with
// this program. If not, see
// <http://www.mongodb.com/licensing/server-side-public-license>.

package fanotify

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"

	"github.com/mutagen-io/mutagen/pkg/sidecar"

	"github.com/mutagen-io/mutagen/sspl/pkg/platform/linux"
)

// Supported indicates whether or not fanotify is supported by the current
// platform version and process capabilities.
var Supported = false

func init() {
	// Verify that we're running in the Mutagen sidecar. At the moment, this is
	// the only environment in which we support fanotify due to its complex API
	// and whole-filesystem watching.
	if !sidecar.EnvironmentIsSidecar() {
		return
	}

	// Verify that the Linux kernel is version 5.1 or later. This is required
	// for fanotify's FAN_REPORT_FID flag (as well as most event flags).
	if major, minor, err := linux.Version(); err != nil {
		return
	} else if major < 5 || (major == 5 && minor < 1) {
		return
	}

	// Verify that the process has sufficient capabilities to use fanotify. It
	// needs CAP_SYS_ADMIN to use the fanotify API and CAP_DAC_READ_SEARCH to
	// use the open_by_handle_at system call. Note that the CAP_* values are bit
	// indices and NOT flags.
	if capabilities, err := linux.Capabilities(); err != nil {
		return
	} else if capabilities&(1<<unix.CAP_SYS_ADMIN) == 0 {
		return
	} else if capabilities&(1<<unix.CAP_DAC_READ_SEARCH) == 0 {
		return
	}

	// At this point, we expect that we'll be able to initialize an fanotify
	// watch, but we're still dependent on CONFIG_FANOTIFY=y having been
	// specified in the kernel build configuration (which is not a given). There
	// isn't a portable and reliable way to get the kernel configuration (and it
	// would be onerous to parse anyway), so we'll just quickly try to create an
	// fanotify watching descriptor to see if the system call succeeds. If
	// fanotify support hasn't been enabled, then we'll get ENOSYS.
	if descriptor, err := unix.FanotifyInit(unix.FAN_REPORT_FID|unix.FAN_CLOEXEC|unix.FAN_NONBLOCK, 0); err != nil {
		return
	} else {
		unix.Close(descriptor)
	}

	// The fanotify API is supported.
	Supported = true
}

// fanotifyEventInfoHeader is a Go port of the Linux fanotify_event_info_header
// structure.
type fanotifyEventInfoHeader struct {
	// infoType is the type of information contained in a
	// fanotify_event_info_fid structure.
	infoType uint8
	// pad is byte alignment padding.
	pad uint8
	// length is the length (in bytes) of the fanotify_event_info_fid structure.
	length uint16
}

// fileHandlePrefix is a Go port of the first two fields of the Linux
// file_handle structure. Since the file_handle structure ends with a
// variable-length opaque object type, we can only include the first two fields
// and then use those to compute a pointer to the opaque file handle.
type fileHandlePrefix struct {
	// bytes is the length (in bytes) of the opaque file handle at the end of
	// the file_handle structure.
	bytes uint32
	// handleType is the type of file handle.
	handleType int32
}

const (
	// fanotifyReadBufferSize is the buffer size for reading fanotify events.
	fanotifyReadBufferSize = 4096

	// pathStale is a sentinel value used to indicate a stale file handle. Since
	// POSIX doesn't allow null bytes in file names, this shouldn't collide with
	// any real path.
	pathStale = "\x00stale"

	// fanotifyEventMetadataSize is the size of unix.FanotifyEventMetadata.
	fanotifyEventMetadataSize = int(unsafe.Sizeof(unix.FanotifyEventMetadata{}))
	// fanotifyEventInfoHeaderSize is the size of fanotifyEventInfoHeader.
	fanotifyEventInfoHeaderSize = int(unsafe.Sizeof(fanotifyEventInfoHeader{}))
	// fsidSize is the size of unix.Fsid.
	fsidSize = int(unsafe.Sizeof(unix.Fsid{}))
	// fileHandlePrefixSize is the size of the fileHandlePrefix structure.
	fileHandlePrefixSize = int(unsafe.Sizeof(fileHandlePrefix{}))
)

// robustOpenByHandleAt is a wrapper around open_by_handle_at that strategically
// retries the operation if ENOMEM is encountered.
func robustOpenByHandleAt(mountFD int, handle unix.FileHandle, flags int) (int, error) {
	// Try a nominal open.
	result, err := unix.OpenByHandleAt(mountFD, handle, flags)
	if err != unix.ENOMEM {
		return result, err
	}

	// We encountered ENOMEM, but generally an immediate retry will succeed, so
	// try again.
	result, err = unix.OpenByHandleAt(mountFD, handle, flags)
	if err != unix.ENOMEM {
		return result, err
	}

	// We've encountered ENOMEM once again. Sleep for a few milliseconds to let
	// the kernel free up memory and then try one last time.
	time.Sleep(5 * time.Millisecond)
	return unix.OpenByHandleAt(mountFD, handle, flags)
}

// processEvent extracts a single event path from an event buffer populated by
// an fanotify watch operating in FAN_REPORT_FID mode. This function will return
// the remaining buffer contents after the event has been processed, as well as
// the path for the processed event. It will return an error in the case of
// overflow detection, invalid data, or path resolution failure. If the event
// file handle is stale, then the buffer will still be advanced and pathStale
// will be returned for the path (with no error).
func processEvent(mountFD int, buffer []byte) ([]byte, string, error) {
	// Ensure that there's enough remaining buffer to account for at least an
	// event metadata structure.
	if len(buffer) < fanotifyEventMetadataSize {
		return nil, "", errors.New("buffer contents too small to contain event metadata")
	}

	// Extract the event metadata, ensure that there's enough remaining buffer
	// to account for all event information, and advance the buffer to point to
	// the event information structure.
	eventMetadata := (*unix.FanotifyEventMetadata)(unsafe.Pointer(&buffer[0]))
	if len(buffer) < int(eventMetadata.Event_len) {
		return nil, "", errors.New("buffer contents too small to contain full event")
	}
	buffer = buffer[fanotifyEventMetadataSize:]

	// Watch for overflow events. In this case there won't be any subsequent
	// event information structure (or at least the fanotify documentation
	// doesn't indicate that there will be).
	if eventMetadata.Mask&unix.FAN_Q_OVERFLOW != 0 {
		return nil, "", ErrWatchInternalOverflow
	}

	// Extract the event information header and verify that the event
	// information is the type that we expect (file information in the form of a
	// fanotify_event_info_fid structure). Then advance the buffer to point to
	// the file_handle portion of the structure.
	eventInfoHeader := (*fanotifyEventInfoHeader)(unsafe.Pointer(&buffer[0]))
	if eventInfoHeader.infoType != unix.FAN_EVENT_INFO_TYPE_FID {
		return nil, "", errors.New("event information with unexpected type")
	}
	buffer = buffer[fanotifyEventInfoHeaderSize+fsidSize:]

	// Extract the file handle. Unfortunately we can't use the structure from
	// the unix package directly because it already performs some tricky type
	// wrapping and adapting. Instead, we have to compute various struct offsets
	// and use the unix.NewFileHandle method to create a viable handle. We leave
	// the buffer pointing to the start of the next event metadata.
	fileHandlePrefix := (*fileHandlePrefix)(unsafe.Pointer(&buffer[0]))
	buffer = buffer[fileHandlePrefixSize:]
	fileHandleBytes := buffer[:fileHandlePrefix.bytes]
	buffer = buffer[fileHandlePrefix.bytes:]
	fileHandle := unix.NewFileHandle(fileHandlePrefix.handleType, fileHandleBytes)

	// Attempt to open the file associated with the event. Note that O_PATH has
	// a different and special meaning in the context of open_by_handle_at (see
	// open_by_handle_at(2)) and (consequently) O_NOFOLLOW is unnecessary.
	//
	// HACK: When rapidly invoking open_by_handle_at, especially in a container,
	// there's a small but non-trivial cross-section for encountering ENOMEM. It
	// generally goes away on a subsequent call, but we have to use this wrapper
	// function to avoid it when we receive rapid notifications.
	eventDescriptor, err := robustOpenByHandleAt(
		mountFD, fileHandle, unix.O_PATH|unix.O_CLOEXEC,
	)
	if err != nil {
		if err == unix.ESTALE {
			return buffer, pathStale, nil
		}
		return nil, "", fmt.Errorf("unable to open event file: %w", err)
	}

	// Read the file path and close the event file.
	path, err := os.Readlink("/proc/self/fd/" + strconv.Itoa(eventDescriptor))
	unix.Close(eventDescriptor)
	if err != nil {
		return nil, "", fmt.Errorf("unable to read event path: %w", err)
	}

	// If this is a deletion and the path has a " (deleted)" suffix, then remove
	// it. This occurs when a file has been unlinked but not yet removed from
	// disk (a strange artifact of fanotify's use of name_to_handle_at and our
	// subsequent call to open_by_handle_at). In this case, the kernel appends
	// " (deleted)" to the end of the file name.
	if eventMetadata.Mask&unix.FAN_DELETE != 0 && strings.HasSuffix(path, " (deleted)") {
		path = path[:len(path)-10]
	}

	// Success.
	return buffer, path, nil
}
