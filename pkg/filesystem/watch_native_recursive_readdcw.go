// +build windows

package filesystem

import (
	"context"
	"os"
	"syscall"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/filesystem/winfsnotify"
)

const (
	// winfsnotifyFlags are the flags to use for recursive winfsnotify watches.
	winfsnotifyFlags = winfsnotify.FS_ALL_EVENTS & ^(winfsnotify.FS_ACCESS | winfsnotify.FS_CLOSE)
)

// watchRootParameters specifies the parameters that should be monitored for
// changes when determining whether or not a watch needs to be re-established.
type watchRootParameters struct {
	// fileAttributes are the Windows file attributes
	fileAttributes uint32
	// creationTime is the creation time.
	creationTime syscall.Filetime
}

// probeWatchRoot probes the parameters of the watch root.
func probeWatchRoot(root string) (watchRootParameters, error) {
	if info, err := os.Lstat(root); err != nil {
		if os.IsNotExist(err) {
			return watchRootParameters{}, err
		}
		return watchRootParameters{}, errors.Wrap(err, "unable to grab root metadata")
	} else if stat, ok := info.Sys().(*syscall.Win32FileAttributeData); !ok {
		return watchRootParameters{}, errors.New("unable to extract raw root metadata")
	} else {
		return watchRootParameters{
			fileAttributes: stat.FileAttributes,
			creationTime:   stat.CreationTime,
		}, nil
	}
}

// watchRootParametersEqual determines whether or not two watch root parameters
// are equal.
func watchRootParametersEqual(first, second watchRootParameters) bool {
	return first.fileAttributes == second.fileAttributes &&
		first.creationTime == second.creationTime
}

type recursiveWatch struct {
	watcher          *winfsnotify.Watcher
	forwardingCancel context.CancelFunc
	eventPaths       chan string
}

func newRecursiveWatch(path string) (*recursiveWatch, error) {
	// Create the watcher.
	watcher, err := winfsnotify.NewWatcher()
	if err != nil {
		return nil, errors.Wrap(err, "unable to create watcher")
	}

	// Create the event paths channel.
	eventPaths := make(chan string, watchEventsBufferSize)

	// Start a cancellable Goroutine to extract and forward paths.
	forwardingContext, forwardingCancel := context.WithCancel(context.Background())
	go func() {
	Forwarding:
		for {
			select {
			case <-forwardingContext.Done():
				break Forwarding
			case e, ok := <-watcher.Event:
				if !ok {
					break Forwarding
				}
				select {
				case eventPaths <- e.Name:
				default:
				}
			}
		}
	}()

	// Start watching.
	if err := watcher.AddWatch(path, winfsnotifyFlags); err != nil {
		forwardingCancel()
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, errors.Wrap(err, "unable to start watching")
	}

	// Done.
	return &recursiveWatch{
		watcher:          watcher,
		forwardingCancel: forwardingCancel,
		eventPaths:       eventPaths,
	}, nil
}

func (w *recursiveWatch) stop() {
	// Stop the underlying event stream.
	// TODO: Should we handle errors here? There's not really anything sane that
	// we can do.
	w.watcher.Close()

	// Cancel forwarding.
	w.forwardingCancel()
}
