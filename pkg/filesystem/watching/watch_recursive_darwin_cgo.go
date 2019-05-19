// +build darwin,cgo

package watching

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/havoc-io/fsevents"
)

const (
	// RecursiveWatchingSupported indicates whether or not the current platform
	// supports native recursive watching.
	RecursiveWatchingSupported = true

	// fseventsChannelCapacity is the capacity to use for the internal FSEvents
	// events channel.
	fseventsChannelCapacity = 50

	// fseventsCoalescingPeriod is the internal latency parameter to use with
	// the FSEvents API. This parameter defines the time window over which
	// multiple events will be coalesced before being delivered from the API in
	// a batch.
	fseventsCoalescingPeriod = 10 * time.Millisecond

	// fseventsFlags are the flags to use for FSEvents watches. The inclusion
	// of the NoDefer (kFSEventStreamCreateFlagNoDefer) flag means that one-shot
	// events that occur outside of a coalescing window will be delivered
	// immediately and then subsequent events will coalesced. This is useful for
	// quick response times on single events without being overwhelmed by
	// rapidly occurring events.
	fseventsFlags = fsevents.NoDefer | fsevents.WatchRoot | fsevents.FileEvents
)

// WatchRecursive performs recursive watching on platforms which support doing
// so natively. The function will stobe the events channel with an empty path
// after the watch has been established. After this function returns, no more
// events will be written to the events channel.
func WatchRecursive(context context.Context, target string, events chan string) error {
	// Ensure that the events channel is buffered.
	if cap(events) < 1 {
		panic("events channel should be buffered")
	}

	// Enforce that the watch target path is absolute. This is necessary because
	// FSEvents will return event paths as absolute paths rooted at the system
	// root (at least with the per-host streams that we're using), and thus
	// we'll need to know the full path to the watch target to make event paths
	// target-relative.
	if !filepath.IsAbs(target) {
		return errors.New("watch target path must be absolute")
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
		return errors.Wrap(err, "unable to resolve symbolic links for watch target")
	} else {
		target = t
	}

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
	// The second race window, which is essentially indefinite and somewhat
	// more philosophical/theoretical, is due to the fact that the unresolved
	// original path provided to this function could diverge in target from
	// what's actually being watched. This is a general problem with watching
	// and not something Mutagen-specific. Fortunately in our case, this
	// divergence essentially never occurs, and even if it does occur, and even
	// if we're relying on native watching to perform fast accurate re-scans, we
	// still have just-in-time checks during transitioning to make sure any
	// changes that we're applying were decided upon based on what's actually on
	// disk at the target location.

	// Create and start the event stream and defer its shutdown.
	rawEvents := make(chan []fsevents.Event, fseventsChannelCapacity)
	eventStream := &fsevents.EventStream{
		Events:  rawEvents,
		Paths:   []string{target},
		Latency: fseventsCoalescingPeriod,
		Flags:   fseventsFlags,
	}
	eventStream.Start()
	defer eventStream.Stop()

	// Strobe the event channel to indicate that watching has started.
	select {
	case events <- "":
	default:
		return errors.New("strobe event overflowed events channel")
	}

	// Loop indefinitely, polling for cancellation, events, and root checks.
	for {
		select {
		case <-context.Done():
			return errors.New("watch cancelled")
		case eventSet, ok := <-rawEvents:
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
					return errors.New("raw events were coalesced")
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

				// Forward the path.
				select {
				case events <- path:
				default:
					return errors.New("event forwarding overflow")
				}
			}
		}
	}
}
