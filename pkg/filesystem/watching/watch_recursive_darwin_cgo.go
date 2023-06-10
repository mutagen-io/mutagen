//go:build darwin && cgo

package watching

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsevents"
)

const (
	// RecursiveWatchingSupported indicates whether or not the current platform
	// supports native recursive watching.
	RecursiveWatchingSupported = true

	// fseventsChannelCapacity is the capacity to use for the internal FSEvents
	// events channel. This doesn't need to be extremely large because (a) we
	// service that channel as fast as the scheduler will allow and (b) FSEvents
	// will perform event coalescing anyway, so each channel entry can store
	// more than one event.
	fseventsChannelCapacity = 50
	// fseventsLatency is the internal latency (coalescing) parameter to use for
	// FSEvents watches. Setting this to a non-zero value allows FSEvents to
	// optimize the transfer of events from kernel to user space by grouping
	// rapid events together. There won't be a significant impact on perceptible
	// latency because we use the kFSEventStreamCreateFlagNoDefer flag to avoid
	// coalescing latency on one-shot events.
	fseventsLatency = 10 * time.Millisecond
	// fseventsFlags are the flags to use for FSEvents watches. The inclusion
	// of the NoDefer (kFSEventStreamCreateFlagNoDefer) flag means that one-shot
	// events that occur outside of a coalescing window will be delivered
	// immediately and then subsequent events will coalesced. This is useful for
	// quick response times on single events without being overwhelmed by
	// rapidly occurring events.
	fseventsFlags = fsevents.NoDefer | fsevents.WatchRoot | fsevents.FileEvents
)

// recursiveWatcher implements RecursiveWatcher using FSEvents.
type recursiveWatcher struct {
	// watch is the underlying FSEvents event stream.
	watch *fsevents.EventStream
	// events is the event delivery channel.
	events chan string
	// errors is the error delivery channel.
	errors chan error
	// cancel is the run loop cancellation function.
	cancel context.CancelFunc
	// done is the run loop completion signaling mechanism.
	done sync.WaitGroup
}

// NewRecursiveWatcher creates a new FSEvents-based recursive watcher using the
// specified target path.
func NewRecursiveWatcher(target string) (RecursiveWatcher, error) {
	// Enforce that the watch target path is absolute. This is necessary because
	// FSEvents will return event paths as absolute paths rooted at the system
	// root (at least with the per-host streams that we're using), and thus
	// we'll need to know the full path to the watch target to make event paths
	// target-relative.
	if !filepath.IsAbs(target) {
		return nil, errors.New("watch target path must be absolute")
	}

	// Fully evaluate any symbolic links in the target. This is necessary
	// because FSEvents will also fully evaluate symbolic links in the watch
	// path provided to it and use that fully evaluated path in any event paths.
	// Thus, if we want to make event paths target-relative, we'll need to know
	// the real target path. Note that, since we know the input path here is
	// absolute, we also know that the output path will be absolute. Also note
	// that calling filepath.EvalSymlinks has the side-effect of enforcing that
	// the target exists.
	if t, err := filepath.EvalSymlinks(target); err != nil {
		return nil, fmt.Errorf("unable to resolve symbolic links for watch target: %w", err)
	} else {
		target = t
	}

	// RACE: There are two race windows with native watching which effectively
	// start here and are worth mentioning:
	//
	// The first is the race window between our symbolic link resolution above
	// and the symbolic link resolution performed by FSEvents on our resolved
	// path when starting its watch. In theory, a component of our resolved path
	// could be replaced by a symbolic link, which would then be further
	// resolved by FSEvents to point elsewhere. In practice, this window is
	// exceptionally small, and a disagreement between our resolution and
	// FSEvents' resolution would manifest as event paths with an unexpected
	// prefix and thus result in an error below.
	//
	// The second race window, which is essentially indefinite and somewhat more
	// philosophical/theoretical, is due to the fact that the unresolved
	// original path provided to this function could diverge in target from
	// what's actually being watched. This is a general problem with watching
	// and not something Mutagen-specific. Fortunately in our case, this
	// divergence essentially never occurs, and even if it does occur, and even
	// if we're relying on native watching to perform fast accurate re-scans, we
	// still have just-in-time checks during transitioning to make sure any
	// changes that we're applying were decided upon based on what's actually on
	// disk at the target location.

	// Create and start the underlying event stream.
	watch := &fsevents.EventStream{
		Events:  make(chan []fsevents.Event, fseventsChannelCapacity),
		Paths:   []string{target},
		Latency: fseventsLatency,
		Flags:   fseventsFlags,
	}
	watch.Start()

	// Create a context to regulate the watcher's run loop.
	ctx, cancel := context.WithCancel(context.Background())

	// Create the watcher.
	watcher := &recursiveWatcher{
		watch:  watch,
		events: make(chan string),
		errors: make(chan error, 1),
		cancel: cancel,
	}

	// Track run loop termination.
	watcher.done.Add(1)

	// Start the run loop.
	go func() {
		watcher.errors <- watcher.run(ctx, target)
		watcher.done.Done()
	}()

	// Success.
	return watcher, nil
}

// run implements the event processing run loop for recursiveWatcher.
func (w *recursiveWatcher) run(ctx context.Context, target string) error {
	// Compute the prefix that we'll need to trim from event paths to make them
	// target-relative (if they aren't the target itself). Since we called
	// filepath.EvalSymlinks above, which calls filepath.Clean, we know that
	// target will be without a trailing slash (unless it's the system root
	// path).
	var eventPathTrimPrefix string
	if target == "/" {
		eventPathTrimPrefix = "/"
	} else {
		eventPathTrimPrefix = target + "/"
	}

	// Perform event forwarding until cancellation or failure.
	for {
		select {
		case <-ctx.Done():
			return ErrWatchTerminated
		case eventSet, ok := <-w.watch.Events:
			// Watch for unexpected event channel closures.
			if !ok {
				return errors.New("internal events channel closed unexpectedly")
			}

			// Process the event set.
			for _, event := range eventSet {
				// Watch for events that would invalidate our watch. The only
				// case that we can ignore is the fsevents.RootChanged
				// (kFSEventStreamEventFlagRootChanged) flag, because FSEvents
				// watches will continue to function across the deletion and
				// recreation of the watch root (or its parent directories). The
				// only case where this doesn't work is when a parent component
				// of the resolved watch target is replaced with a symbolic
				// link, but this is a subset of the second race condition
				// described above (target divergence) and something that we
				// can't do much about in general.
				if event.Flags&fsevents.MustScanSubDirs != 0 {
					return ErrWatchInternalOverflow
				} else if event.Flags&fsevents.Mount != 0 {
					return errors.New("volume mounted under watch root")
				} else if event.Flags&fsevents.Unmount != 0 {
					return errors.New("volume unmounted under watch root")
				}

				// Convert the event path to be target-relative.
				path := event.Path
				if path == target {
					path = ""
				} else if strings.HasPrefix(path, eventPathTrimPrefix) {
					path = path[len(eventPathTrimPrefix):]
				} else {
					return errors.New("event path is not watch target and does not have expected prefix")
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
}

// Events implements RecursiveWatcher.Events.
func (w *recursiveWatcher) Events() <-chan string {
	return w.events
}

// Errors implements RecursiveWatcher.Errors.
func (w *recursiveWatcher) Errors() <-chan error {
	return w.errors
}

// Terminate implements RecursiveWatcher.Terminate.
func (w *recursiveWatcher) Terminate() error {
	// Signal termination.
	w.cancel()

	// Wait for the run loop to exit.
	w.done.Wait()

	// Terminate the underlying event stream.
	w.watch.Stop()

	// Success.
	return nil
}
