package local

import (
	"github.com/mutagen-io/mutagen/pkg/synchronization/core"
	"github.com/mutagen-io/mutagen/pkg/synchronization/rsync"
)

// stager is the interface implemented by file staging implementations. Its
// Contains, Sink, and Provide methods must be safe for concurrent invocation,
// as well as for concurrent usage with the io.WriteCloser instances returned by
// Sink. However, its Initialize and Finalize methods need not be safe for
// concurrent invocation.
type stager interface {
	// Initialize informs the stager that staging and transitioning are about to
	// commence. This method will always be called before any calls to Contains,
	// Sink, or Provide, even after a previously failed staging session for
	// which Finalize was not invoked.
	Initialize() error
	// Contains returns whether or not the stager contains the specified
	// content.
	Contains(path string, digest []byte) (bool, error)
	// Sinker is the interface that the stager must implement to receive files
	// over an rsync transmission stream.
	rsync.Sinker
	// Provider is the interface that the stager must implement to provide files
	// for transition operations after staging is complete.
	core.Provider
	// Finalize informs the stager that staging has completed and that no
	// further Contains, Sink, or Provide calls will be made until after the
	// next call to Initialize. Implementations should use this method to clean
	// up any on-disk resources and may use this method to free any in-memory
	// resources.
	Finalize() error
}
