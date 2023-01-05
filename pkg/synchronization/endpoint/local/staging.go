package local

import (
	"github.com/mutagen-io/mutagen/pkg/synchronization/core"
	"github.com/mutagen-io/mutagen/pkg/synchronization/rsync"
)

// stager is the interface implemented by local file staging implementations.
type stager interface {
	// Prepare informs the stager that staging and transitioning are about to
	// commence. This method will always be called before any calls to Sink or
	// Provide, even after a previously failed staging session after which
	// Finalize was not invoked.
	Prepare() error
	// Sinker is the interface that the stager must implement to receive files
	// over an rsync transmission stream. The stager interface adds the
	// additional requirement that no other methods on the stager may be invoked
	// until the Close method of the io.WriteCloser returned by Sink has been
	// invoked.
	rsync.Sinker
	// Provider is the interface that the stager must implement to provide files
	// for transition operations after staging is complete.
	core.Provider
	// Finalize informs the stager that staging has completed and that no
	// further Sink or Provide calls will be made until after the next call to
	// Prepare. Implementations should use this method to clean up any on-disk
	// resources and may use this method to free any in-memory resources.
	Finalize() error
}
