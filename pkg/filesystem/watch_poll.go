package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

const (
	// defaultInitialContentMapCapacity is the default content map capacity if
	// the existing content map is empty.
	defaultInitialContentMapCapacity = 1024
)

// fileInfoEqual determines whether or not file metadata is equivalent (i.e.
// whether or not a change notification is warranted).
func fileInfoEqual(first, second os.FileInfo) bool {
	// Compare modes.
	if first.Mode() != second.Mode() {
		return false
	}

	// If we're dealing with directories, don't check size or time. Size doesn't
	// really make sense and modification time will be affected by our
	// executability preservation or Unicode decomposition probe file creation.
	if first.IsDir() {
		return true
	}

	// Compare size and time.
	return first.Size() == second.Size() &&
		first.ModTime().Equal(second.ModTime())
}

// poll creates a simple snapshot of a synchronization root, detecting (and
// optionally tracking) changes from an existing snapshot.
func poll(root string, existing map[string]os.FileInfo, trackChanges bool) (map[string]os.FileInfo, bool, map[string]bool, error) {
	// Create our result map.
	initialContentMapCapacity := len(existing)
	if initialContentMapCapacity == 0 {
		initialContentMapCapacity = defaultInitialContentMapCapacity
	}
	contents := make(map[string]os.FileInfo, initialContentMapCapacity)

	// Create our change tracking map (only allocating if we're actually
	// tracking changes).
	var changes map[string]bool
	if trackChanges {
		// TODO: Should we use an initial capacity here? It'll be small or empty
		// most of the time, but we'll have a large allocation penalty on the
		// first scan. Perhaps we can use a non-zero initial capacity if
		// len(existing) == 0?
		changes = make(map[string]bool)
	}

	// Create a walk visitor.
	changed := false
	rootDoesNotExist := false
	visitor := func(path string, info os.FileInfo, err error) error {
		// Handle walk error cases.
		if err != nil {
			// If we're at the root and this is a non-existence error, then we
			// can create a valid result (and empty map) as well as determine
			// whether or not there's been a change.
			if path == root && os.IsNotExist(err) {
				changed = len(existing) > 0
				rootDoesNotExist = true
				return err
			}

			// If this is a non-root non-existence error, then something was
			// seen during the directory listing and then failed the stat call.
			// This is a sign of concurrent deletion, so just ignore this file.
			// Our later checks will determine if this was concurent deletion of
			// a file we're meant to be watching.
			if os.IsNotExist(err) {
				return nil
			}

			// Other errors are more problematic.
			return err
		}

		// If this is an intermediate temporary file, then ignore it.
		if IsTemporaryFileName(filepath.Base(path)) {
			return nil
		}

		// Insert the entry for this path.
		contents[path] = info

		// Compare the entry for this path.
		pathChanged := false
		if previous, ok := existing[path]; !ok || !fileInfoEqual(info, previous) {
			pathChanged = true
		}

		// Update change tracker.
		if pathChanged {
			changed = true
		}

		// If we're tracking changes and this path changed, register it. We
		// always track the parent of the path, and for directories we also
		// track the path itself, since these are where changes are going to be
		// visible.
		if trackChanges && pathChanged {
			if info.IsDir() {
				changes[path] = true
			}
			if path != root {
				changes[filepath.Dir(path)] = true
			}
		}

		// Success.
		return nil
	}

	// Perform the walk. If it fails, and it's not due to the root not existing,
	// then we can't return a valid result and need to abort.
	if err := Walk(root, visitor); err != nil && !rootDoesNotExist {
		return nil, false, nil, errors.Wrap(err, "unable to perform filesystem walk")
	}

	// If the length of the result map has changed, then there's been a change.
	// This could be due to files being deleted.
	if len(contents) != len(existing) {
		changed = true
	}

	// Done.
	return contents, changed, changes, nil
}

// watchPoll performs poll-based filesystem watching.
func watchPoll(context context.Context, root string, events chan struct{}, pollInterval uint32) error {
	// Validate the polling interval and convert it to a duration.
	if pollInterval == 0 {
		return errors.New("polling interval must be greater than 0 seconds")
	}
	pollIntervalDuration := time.Duration(pollInterval) * time.Second

	// Create a timer to regulate polling. Start it with a 0 duration so that
	// the first polling takes place immediately. Subsequent pollings will take
	// place at the normal interval.
	timer := time.NewTimer(0)

	// Loop and poll for changes, but watch for cancellation.
	var contents map[string]os.FileInfo
	for {
		select {
		case <-context.Done():
			// Abort the watch.
			return errors.New("watch cancelled")
		case <-timer.C:
			// Perform a scan. If there's an error or no change, then reset the
			// timer and try again. We have to assume that errors here are due
			// to concurrent modifications, so there's not much we can do to
			// handle them.
			newContents, changed, _, err := poll(root, contents, false)
			if err != nil || !changed {
				timer.Reset(pollIntervalDuration)
				continue
			}

			// Store the new contents.
			contents = newContents

			// Forward the event in a non-blocking fashion.
			select {
			case events <- struct{}{}:
			default:
			}

			// Reset the timer and continue polling.
			timer.Reset(pollIntervalDuration)
		}
	}
}
