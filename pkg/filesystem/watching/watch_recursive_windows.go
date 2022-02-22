package watching

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/mutagen-io/mutagen/pkg/filesystem/watching/internal/third_party/winfsnotify"
)

const (
	// RecursiveWatchingSupported indicates whether or not the current platform
	// supports native recursive watching.
	RecursiveWatchingSupported = true

	// watchRootMetadataPollingInterval is the interval at which the watch root
	// will be checked for changes.
	watchRootMetadataPollingInterval = 5 * time.Second

	// winfsnotifyFlags are the flags to use for winfsnotify watches.
	winfsnotifyFlags = winfsnotify.FS_ALL_EVENTS & ^(winfsnotify.FS_ACCESS | winfsnotify.FS_CLOSE)
)

// watchRootParametersEqual determines whether or not the metadata for a path
// being used as a watch root has changed sufficiently to warrant recreating the
// watch.
func watchRootParametersEqual(first, second os.FileInfo) bool {
	// Extract the underlying metadata.
	firstData, firstOk := first.Sys().(*syscall.Win32FileAttributeData)
	secondData, secondOk := second.Sys().(*syscall.Win32FileAttributeData)

	// Check for equality.
	return firstOk && secondOk &&
		firstData.FileAttributes == secondData.FileAttributes &&
		firstData.CreationTime == secondData.CreationTime
}

// recursiveWatcher implements RecursiveWatcher using ReadDirectoryChangesW.
type recursiveWatcher struct {
	// watch is the underlying ReadDirectoryChangesW-based watcher.
	watch *winfsnotify.Watcher
	// events is the event delivery channel.
	events chan map[string]bool
	// errors is the error delivery channel.
	errors chan error
	// cancel is the run loop cancellation function.
	cancel context.CancelFunc
	// done is the run loop completion signaling mechanism.
	done chan struct{}
}

// NewRecursiveWatcher creates a new FSEvents-based recursive watcher using the
// specified target path.
func NewRecursiveWatcher(target string) (RecursiveWatcher, error) {
	// Resolve any symbolic links in the watch target. This is necessary because
	// we're using the parent directory of the target path as the watch root and
	// ReadDirectoryChangesW doesn't watch across symbolic link boundaries, so
	// if the target leaf is a symbolic link, we won't see any changes inside of
	// it. It's worth noting that intermediate symbolic links aren't really a
	// problem (and their unresolved form will even be used as the prefix for
	// generated events), so in theory we might just be able to resolve the leaf
	// component (if it's a symbolic link), but it's easier just to call
	// filepath.EvalSymlinks. Note that calling filepath.EvalSymlinks has the
	// side-effect of enforcing that the target exists.
	if t, err := filepath.EvalSymlinks(target); err != nil {
		return nil, fmt.Errorf("unable to resolve symbolic links for watch target: %w", err)
	} else {
		target = t
	}

	// Enforce that the watch target path is valid for passing to filepath.Dir.
	// We take a conservative approach here, effectively requiring that the
	// path has the format VolumeName + "\" + .... The reason we don't use
	// filepath.IsAbs here is that, on Windows, it also treats reserved names as
	// absolute. Note that, since we called filepath.EvalSymlinks above, and it
	// calls filepath.Clean, we know that any slashes in target at this point
	// will be backslashes.
	volumeName := filepath.VolumeName(target)
	if volumeName == "" {
		return nil, errors.New("resolved target missing volume name")
	} else if len(target) <= len(volumeName) {
		return nil, errors.New("target shorter than or composed only of volume name")
	} else if target[len(volumeName)] != '\\' {
		return nil, errors.New("resolved target has invalid format")
	}

	// Compute the watch root, which on Windows will be the parent of the watch
	// target.
	watchRoot := filepath.Dir(target)

	// Query the initial watch root metadata. Note that we use os.Stat because
	// we want to follow the same resolution behavior as CreateFileW (which is
	// called without FILE_FLAG_OPEN_REPARSE_POINT in the watcher).
	initialWatchRootMetadata, err := os.Stat(watchRoot)
	if err != nil {
		return nil, fmt.Errorf("unable to query initial watch root metadata: %w", err)
	}

	// RACE: There are three race windows with native watching which effectively
	// start here and are worth mentioning:
	//
	// The first is the race window between our symbolic link resolution above
	// and the symbolic link resolution performed by CreateFileW on our resolved
	// path when opening the directory to watch. In theory, a component of our
	// resolved path could be replaced by a symbolic link, which would then be
	// further resolved by CreateFileW to point elsewhere. In practice, this
	// window is exceptionally small, and a disagreement between our resolution
	// and the location resolved by CreateFileW would be picked up by our watch
	// root polling.
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

	// Create the underlying watcher and add the watch.
	watch, err := winfsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("unable to create watcher: %w", err)
	} else if err = watch.AddWatch(watchRoot, winfsnotifyFlags); err != nil {
		return nil, fmt.Errorf("unable to start watching: %w", err)
	}

	// Create a context to regulate the watcher's run loop.
	ctx, cancel := context.WithCancel(context.Background())

	// Create the watcher.
	watcher := &recursiveWatcher{
		watch:  watch,
		events: make(chan map[string]bool),
		errors: make(chan error, 1),
		cancel: cancel,
		done:   make(chan struct{}),
	}

	// Start the run loop.
	go func() {
		watcher.errors <- watcher.run(ctx, watchRoot, initialWatchRootMetadata, target)
	}()

	// Success.
	return watcher, nil
}

// run implements the event processing run loop for recursiveWatcher.
func (w *recursiveWatcher) run(ctx context.Context, watchRoot string, initialWatchRootMetadata os.FileInfo, target string) error {
	// Signal completion when done.
	defer close(w.done)

	// Compute the prefix that we'll use to (a) filter events to those occurring
	// at or under the target and (b) trim off to make paths target-relative
	// (assuming they aren't the target itself). Note that filepath.EvalSymlinks
	// calls filepath.Clean, so target will be without a trailing slash (unless
	// it's a drive root, in which case it will have a trailing slash that's
	// guaranteed (by filepath.Clean) to be a backslash). We also know that
	// target will be non-empty at this point.
	var eventPathTrimPrefix string
	if target[len(target)-1] == '\\' {
		eventPathTrimPrefix = target
	} else {
		eventPathTrimPrefix = target + "\\"
	}

	// Create a timer to watch for changes to the watch root. We start this
	// timer with a 0 duration so that the first check takes place immediately.
	// Subsequent checks will take place at the normal interval. We defer a stop
	// operation to ensure that it's not running when we return.
	watchRootCheckTimer := time.NewTimer(0)
	defer watchRootCheckTimer.Stop()

	// Create a coalescing timer, initially stopped and drained, and ensure that
	// it's stopped once we return.
	coalescingTimer := time.NewTimer(0)
	if !coalescingTimer.Stop() {
		<-coalescingTimer.C
	}
	defer coalescingTimer.Stop()

	// Create an empty pending event.
	pending := make(map[string]bool)

	// Create a separate channel variable to track the target events channel. We
	// keep it nil to block transmission until the pending event is non-empty
	// and the coalescing timer has fired.
	var pendingTarget chan<- map[string]bool

	// Loop indefinitely, polling for cancellation, events, and root checks.
	for {
		select {
		case <-ctx.Done():
			return ErrWatchTerminated
		case event, ok := <-w.watch.Event:
			// Watch for unexpected event channel closures.
			if !ok {
				return errors.New("internal events channel closed unexpectedly")
			}

			// Watch for event overflows that would invalidate our watch.
			if event.Mask == winfsnotify.FS_Q_OVERFLOW {
				return errors.New("internal event overflow")
			}

			// Extract the path.
			path := event.Name

			// Convert the event path to be target-relative and replace
			// backslashes with forward slashes. If the path isn't the target or
			// a child of the target, then we ignore it.
			if path == target {
				path = ""
			} else if strings.HasPrefix(path, eventPathTrimPrefix) {
				path = path[len(eventPathTrimPrefix):]
				path = strings.ReplaceAll(path, "\\", "/")
			} else {
				continue
			}

			// Record the path.
			pending[path] = true

			// Check if we've exceeded the maximum number of allowed pending
			// paths. We're technically allowing ourselves to go one over the
			// limit here, but to avoid that we'd have to check whether or not
			// each path was already in pending before adding it, and that would
			// be expensive. Since this is a purely internal check for the
			// purpose of avoiding excessive memory usage, this small transient
			// overflow is fine.
			if len(pending) > watchCoalescingMaximumPendingPaths {
				return ErrTooManyPendingPaths
			}

			// We may have already had a pending event that was coalesced and
			// ready to be delivered, but now we've seen more changes and we're
			// going to create a new coalescing window, so we'll block event
			// transmission until the new coalescing window is complete.
			pendingTarget = nil

			// Reset the coalesing timer. We don't know if it was already
			// running, so we need to drain it in a non-blocking fashion.
			if !coalescingTimer.Stop() {
				select {
				case <-coalescingTimer.C:
				default:
				}
			}
			coalescingTimer.Reset(watchCoalescingWindow)
		case <-watchRootCheckTimer.C:
			// Grab the current watch root parameters. Note that we continue to
			// use os.Stat for the reasons outlined above.
			currentWatchRootMetadata, err := os.Stat(watchRoot)
			if err != nil {
				return fmt.Errorf("unable to query watch root metadata: %w", err)
			}

			// Abort watching if the watch has been invalidated.
			if !watchRootParametersEqual(initialWatchRootMetadata, currentWatchRootMetadata) {
				return errors.New("watch root change")
			}

			// Reset the timer and continue watching.
			watchRootCheckTimer.Reset(watchRootMetadataPollingInterval)
		case <-coalescingTimer.C:
			// Set the target events channel to the actual events channel.
			pendingTarget = w.events
		case pendingTarget <- pending:
			// Create a new pending event.
			pending = make(map[string]bool)

			// Block event transmission until the event is non-empty.
			pendingTarget = nil
		}
	}
}

// Events implements RecursiveWatcher.Events.
func (w *recursiveWatcher) Events() <-chan map[string]bool {
	return w.events
}

// Errors implements RecursiveWatcher.Errors.
func (w *recursiveWatcher) Errors() <-chan error {
	return w.errors
}

// Terminate implements RecursiveWatcher.Terminate.
func (w *recursiveWatcher) Terminate() error {
	// Signal cancellation.
	w.cancel()

	// Wait for the run loop to exit.
	<-w.done

	// Terminate the underlying watcher.
	return w.watch.Close()
}
