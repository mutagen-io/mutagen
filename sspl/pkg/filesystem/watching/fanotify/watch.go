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
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/sys/unix"

	"github.com/mutagen-io/mutagen/pkg/sidecar"
)

var (
	// ErrWatchInternalOverflow indicates that a watcher saw an event buffering
	// overflow in its underlying watching mechanism.
	ErrWatchInternalOverflow = errors.New("internal event overflow")
	// ErrWatchTerminated indicates that a watcher has been terminated.
	ErrWatchTerminated = errors.New("watch terminated")
)

// RecursiveWatcher implements watching.RecursiveWatcher using fanotify.
type RecursiveWatcher struct {
	// watch is a handle for closing the underlying fanotify watch descriptor.
	watch io.Closer
	// events is the event delivery channel.
	events chan string
	// writeErrorOnce ensures that only one error is written to errors.
	writeErrorOnce sync.Once
	// errors is the error delivery channel.
	errors chan error
	// cancel is the run loop cancellation function.
	cancel context.CancelFunc
	// done is the run loop completion signaling mechanism.
	done sync.WaitGroup
}

// NewRecursiveWatcher creates a new fanotify-based recursive watcher using the
// specified target path.
func NewRecursiveWatcher(target string) (*RecursiveWatcher, error) {
	// Enforce that the watch target path is absolute. This is necessary for our
	// invocation of fanotify_mark and to adjust incoming event paths to be
	// relative to the watch target.
	if !filepath.IsAbs(target) {
		return nil, errors.New("watch target path must be absolute")
	}

	// TODO: It's unclear if we want to perform symbolic link evaluation here
	// like we do in other watchers. If we do, then we may want to make
	// adjustments to our open and fanotify_mark calls below. At the moment,
	// it's irrelevant to our very controlled use case. If we do perform
	// symbolic link evaluation, then we can remove the filepath.Clean call.

	// Ensure that the target is cleaned. This is necessary for adjusting event
	// paths to be target-relative.
	target = filepath.Clean(target)

	// Determine the mount point for the path.
	mountPoint := sidecar.VolumeMountPointForPath(target)
	if mountPoint == "" {
		return nil, errors.New("path does not exist at or below a mount point")
	}

	// Get a file descriptor for the mount point to use with open_by_handle_at.
	// Despite the claim in the Linux open(2) man page, it doesn't appear that
	// O_PATH returns a file descriptor suitable for use with all *at functions.
	// Specifically, it doesn't work with open_by_handle_at, and thus we need to
	// perform a "full" open operation.
	mountDescriptor, err := unix.Open(
		mountPoint,
		unix.O_RDONLY|unix.O_CLOEXEC|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to open mount point: %w", err)
	}

	// Create an fanotify watch capable of detecting file events (i.e. using
	// FAN_REPORT_FID). We set the descriptor for this watch to be non-blocking
	// so that we can use an os.File to poll on it and enable read cancellation.
	// The fanotify documentation explicitly states that this descriptor is
	// compatible with epoll, poll, and select, so we know it will work with the
	// Go poller. Also, because we're using FAN_REPORT_FID, we won't receive
	// file descriptors automatically and thus don't need to provide flags for
	// their construction.
	watchDescriptor, err := unix.FanotifyInit(unix.FAN_REPORT_FID|unix.FAN_CLOEXEC|unix.FAN_NONBLOCK, 0)
	if err != nil {
		unix.Close(mountDescriptor)
		return nil, fmt.Errorf("unable to initialize fanotify: %w", err)
	}

	// Add the target path to the watch. We notably exclude FAN_DELETE_SELF from
	// our event mask because it is (almost) always accompanied by a stale file
	// handle from which we cannot obtain a corresponding path. In rare cases
	// (that come down to race conditions), it is possible to open a deleted
	// file by its handle before it is fully removed from the filesystem, but
	// the path will have " (deleted)" appended by the kernel. In any case, we
	// don't need the deleted path when using accelerated scanning, only its
	// parent path (which will come from the FAN_DELETE event). However, we do
	// need FAN_MOVE_SELF, because FAN_MOVE alone will only tell accelerated
	// scanning to rescan the parent, which in the case of file replacement on
	// move wouldn't detect changes to the replaced file. We also exclude
	// FAN_CLOSE_WRITE because it is generated in conjunction with a FAN_MODIFY
	// event (which we already watch for) if changes are flushed to disk on
	// closure, so we can skip it and avoid spurious events generated by closure
	// of unmodified writable files. Also, despite what the documentation says,
	// FAN_Q_OVERFLOW should not be specified as part of this mask, otherwise
	// EINVAL will be returned. The generation of overflow events is automatic.
	if err := unix.FanotifyMark(
		watchDescriptor,
		unix.FAN_MARK_ADD|unix.FAN_MARK_FILESYSTEM|unix.FAN_MARK_ONLYDIR|unix.FAN_MARK_DONT_FOLLOW,
		unix.FAN_CREATE|unix.FAN_MOVE|unix.FAN_MODIFY|unix.FAN_ATTRIB|unix.FAN_DELETE|
			unix.FAN_ONDIR,
		-1, mountPoint,
	); err != nil {
		unix.Close(watchDescriptor)
		unix.Close(mountDescriptor)
		return nil, fmt.Errorf("unable to establish fanotify watch: %w", err)
	}

	// Convert the watch descriptor to an os.File so that it's pollable.
	watch := os.NewFile(uintptr(watchDescriptor), "fanotify")

	// Create a context to regulate the watcher's polling and run loops.
	ctx, cancel := context.WithCancel(context.Background())

	// Create the watcher.
	watcher := &RecursiveWatcher{
		watch:  watch,
		events: make(chan string),
		errors: make(chan error, 1),
		cancel: cancel,
	}

	// Track run loop termination.
	watcher.done.Add(1)

	// Start the run loop.
	go func() {
		err := watcher.run(ctx, watch, mountDescriptor, target)
		unix.Close(mountDescriptor)
		watcher.writeErrorOnce.Do(func() {
			watcher.errors <- err
		})
		watcher.done.Done()
	}()

	// Success.
	return watcher, nil
}

// run implements the event processing run loop for RecursiveWatcher.
func (w *RecursiveWatcher) run(ctx context.Context, watch io.Reader, mountDescriptor int, target string) error {
	// Compute the prefix that we'll need to trim from event paths to make them
	// target-relative (if they aren't the target itself). We know that target
	// will be clean, and thus lacking a trailing slash (unless it's the system
	// root path).
	var eventPathTrimPrefix string
	if target == "/" {
		eventPathTrimPrefix = "/"
	} else {
		eventPathTrimPrefix = target + "/"
	}

	// Loop until cancellation or a read error occurs.
	var buffer [fanotifyReadBufferSize]byte
	for {
		// Read the next group of events. Note that the fanotify API will only
		// return whole events into the buffer, so there's no need to worry
		// about partial event reads.
		read, err := watch.Read(buffer[:])
		if err != nil {
			return fmt.Errorf("unable to read from fanotify watch: %w", err)
		}

		// Process the events.
		populated := buffer[:read]
		for len(populated) > 0 {
			// Process a single event.
			remaining, path, err := processEvent(mountDescriptor, populated)
			if err != nil {
				if err == ErrWatchInternalOverflow {
					return err
				}
				return fmt.Errorf("unable to extract event path: %w", err)
			}
			populated = remaining

			// If the path was stale, then just ignore it.
			if path == pathStale {
				continue
			}

			// Convert the event path to be target-relative. We have to ignore
			// anything that doesn't fall at or below our watch target for two
			// reasons:
			//
			// First, our watch and path resolution location is the mount point,
			// not necessarily the watch target. Much like on Windows, we have
			// to watch outside of the target in order to ensure a stable watch
			// and to ensure that we're seeing changes to the target itself
			// (though, in this case, we do it more for stability since we know
			// the volume mounts aren't likely to disappear).
			//
			// Second, with fanotify, we're watching an entire filesystem, and
			// thus we may see events that occur outside of the mount point that
			// we're watching, especially if that mount point isn't mounting the
			// filesystem root. If the event can't be referenced by a path
			// beneath the mount point that we're using with open_by_handle_at,
			// then a read of its path will result in "/". For example, watching
			// container volumes that are just bind-mounted directories from the
			// host filesystem will result in a watch of the entire host
			// filesystem, but a modification to a path outside of the volume
			// directory on the host filesystem will yield a notification with
			// a path resolving to "/" (when resolved relative to the mount
			// point). This is basically designed to indicate that the event
			// occurred outside the mount point. Fortunately, as long as the
			// target location can be referenced by a path beneath the mount
			// point being used as the reference point for open_by_handle_at, a
			// valid path will be returned. This means (e.g.) that a single
			// volume mounted in multiple containers (even at different mount
			// points) will still yield event notifications with paths relative
			// to the mount point within the container performing the watch.
			if path == target {
				path = ""
			} else if strings.HasPrefix(path, eventPathTrimPrefix) {
				path = path[len(eventPathTrimPrefix):]
			} else {
				continue
			}

			// Transmit the path.
			select {
			case w.events <- path:
			case <-ctx.Done():
				return ErrWatchTerminated
			}
		}
	}
}

// Events implements filesystem/watching.RecursiveWatcher.Events.
func (w *RecursiveWatcher) Events() <-chan string {
	return w.events
}

// Errors implements filesystem/watching.RecursiveWatcher.Errors.
func (w *RecursiveWatcher) Errors() <-chan error {
	return w.errors
}

// Terminate implements filesystem/watching.RecursiveWatcher.Terminate.
func (w *RecursiveWatcher) Terminate() error {
	// Write a termination error to the errors channel since we're going to
	// close the watch and we don't want a read error in the run loop to appear
	// to the consumer if it's simply due to termination.
	w.writeErrorOnce.Do(func() {
		w.errors <- ErrWatchTerminated
	})

	// Signal termination to run loop. The run loop can block in multiple ways
	// and thus needs both watch closure and context cancellation to signal
	// termination. The mount descriptor used for resolving paths is closed by
	// the run loop Goroutine when it exits.
	err := w.watch.Close()
	w.cancel()

	// Wait for the run loop to exit.
	w.done.Wait()

	// Done.
	return err
}
