package synchronization

import (
	"context"

	"github.com/mutagen-io/mutagen/pkg/synchronization/core"
	"github.com/mutagen-io/mutagen/pkg/synchronization/rsync"
)

// Endpoint defines the interface to which synchronization endpoints must
// adhere for a single session. It provides all primitives necessary to support
// synchronization. None of its methods should be considered safe for concurrent
// invocation except Shutdown. If any method returns an error, the endpoint
// should be considered failed and no more of its methods (other than Shutdown)
// should be invoked.
type Endpoint interface {
	// Poll performs a one-shot polling operation for filesystem modifications
	// in the endpoint's root. It blocks until either an event occurs, the
	// provided context is cancelled, or an error occurs. In the first two cases
	// it returns nil and in the latter case it returns the error that occurred.
	Poll(ctx context.Context) error

	// Scan performs a scan of the endpoint's synchronization root. If a non-nil
	// ancestor is passed, then it will be used as a baseline for a deltified
	// snapshot transfer if the endpoint is remote. The ancestor may be nil, in
	// which case the transfer of the initial snapshot may be less than optimal.
	// The full parameter forces the function to perform a full (but still warm)
	// scan, avoiding any acceleration that might be available on the endpoint.
	// The function returns the scan result, any error that occurred while
	// trying to perform the scan, and a boolean indicating whether or not to
	// re-try the scan if an error occurred. Any non-fatal problems encountered
	// during the scan can be extracted from the resulting content.
	Scan(ctx context.Context, ancestor *core.Entry, full bool) (*core.Snapshot, error, bool)

	// Stage performs file staging on the endpoint. It accepts a list of file
	// paths and a separate list of desired digests corresponding to those
	// paths. If these lists do not have the same length, this method should
	// return an error. For optimal performance, the paths should be passed in
	// depth-first traversal order. This method will filter the list of required
	// paths based on what is already available from previously interrupted
	// staging operations and what can be staged directly from the endpoint
	// filesystem (e.g. in cases of renames and copies), and then return a list
	// of paths, their respective signatures, and a receiver to receive them.
	// The returned path list should maintain depth-first traversal ordering for
	// its filtered paths, again for performance reasons. If the list of paths
	// is empty (and the error non-nil), then all paths were either already
	// staged or able to be staged from the endpoint filesystem, and the
	// receiver must be nil. Otherwise, the receiver must be non-nil and must be
	// finalized (i.e. transmitted to) before subsequent methods can be invoked
	// on the endpoint. This method is allowed to modify the provided argument
	// slices. If the returned receiver fails, the endpoint should be considered
	// tainted and not used (though shutdown can and should still be invoked).
	Stage(paths []string, digests [][]byte) ([]string, []*rsync.Signature, rsync.Receiver, error)

	// Supply transmits files in a streaming fashion using the rsync algorithm
	// to the specified receiver.
	Supply(paths []string, signatures []*rsync.Signature, receiver rsync.Receiver) error

	// Transition performs the specified transitions on the endpoint. It returns
	// the respective results of the specified change operations, a list of
	// non-fatal problems encountered during the transition operation, a boolean
	// indicating whether or not the endpoint was missing staged files, and any
	// error occurred while trying to perform the transition operation.
	// TODO: Should we consider pre-emptability for transition? It could
	// probably be done by just checking for cancellation during each transition
	// path and reporting "cancelled" for problems arising after that, but
	// usually the long-blocking transitions are going to be the ones where
	// we're creating the root with a huge number of files and wouldn't catch
	// cancellation until they're all done anyway.
	Transition(ctx context.Context, transitions []*core.Change) ([]*core.Entry, []*core.Problem, bool, error)

	// Shutdown terminates any resources associated with the endpoint. For local
	// endpoints, Shutdown will not preempt calls, but for remote endpoints it
	// will because it closes the underlying connection to the endpoint
	// (actually, it terminates that connection). Shutdown can safely be called
	// concurrently with other methods, though it's only recommended when you
	// don't want the possibility of preempting the method (e.g. in Transition)
	// or you know that the operation can continue and terminate on its own
	// (e.g. in Scan). Shutdown should only be invoked once.
	Shutdown() error
}
