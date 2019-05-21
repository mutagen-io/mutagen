package session

import (
	"context"

	"github.com/havoc-io/mutagen/pkg/rsync"
	"github.com/havoc-io/mutagen/pkg/sync"
)

// Endpoint defines the interface to which synchronization endpoints must
// adhere for a single session. It provides all primitives necessary to support
// synchronization. None of its methods should be considered safe for concurrent
// invocation except Shutdown. If any method returns an error, the endpoint
// should be considered failed and no more of its methods (other than Shutdown)
// should be invoked.
type Endpoint interface {
	// Poll performs a one-shot poll for filesystem modifications in the
	// endpoint's root. It blocks until an event occurs, the provided context is
	// cancelled, or an error occurs. In the first two cases it returns nil. The
	// provided context is guaranteed to be cancelled eventually.
	Poll(context context.Context) error

	// Scan performs a scan of the endpoint's synchronization root. It requires
	// the ancestor to be passed in for optimized snapshot transfers if the
	// endpoint is remote. The ancestor may be nil, in which transfers from
	// remote endpoints may be less than optimal. The full parameter forces the
	// function to perform a full (warm) scan, avoiding any acceleration that
	// might be available on the endpoint. The function returns the scan result,
	// a boolean indicating whether or not the synchronization root preserves
	// POSIX executability bits, any error that occurred while trying to create
	// the scan, and a boolean indicating whether or not to re-try the scan (in
	// the event of an error).
	Scan(ancestor *sync.Entry, full bool) (*sync.Entry, bool, error, bool)

	// Stage performs staging on the endpoint. It accepts a list of file paths
	// and a separate list of desired digests corresponding to those paths. For
	// performance reasons, the paths should be passed in depth-first traversal
	// order. This method will filter the list based on what it already has
	// staged from previously interrupted stagings and what can be staged from
	// local contents (e.g. in cases of renames and copies), and then return a
	// list of paths, their signatures, and a receiver to receive them. The
	// returned path list must maintain relative ordering for its filtered
	// paths, again for performance reasons. If the list of paths is empty, then
	// all paths were either already staged or able to be staged from local
	// data, and the receiver will be nil. Otherwise, the receiver will be
	// non-nil and must be finalized (i.e. transmitted to) before subsequent
	// methods can be invoked on the endpoint. This method is allowed to modify
	// the argument slices. If the receiver fails, the endpoint should be
	// considered contaminated and not used (though shutdown can and should
	// still be invoked).
	Stage(paths []string, digests [][]byte) ([]string, []*rsync.Signature, rsync.Receiver, error)

	// Supply transmits files in a streaming fashion using the rsync algorithm
	// to the specified receiver.
	Supply(paths []string, signatures []*rsync.Signature, receiver rsync.Receiver) error

	// Transition performs the specified transitions on the endpoint. It returns
	// a list of successfully applied changes and a list of problems that
	// occurred while applying transitions.
	// TODO: Should we consider pre-emptability for transition? It could
	// probably be done by just checking for cancellation during each transition
	// path and reporting "cancelled" for problems arising after that, but
	// usually the long-blocking transitions are going to be the ones where
	// we're creating the root with a huge number of files and wouldn't catch
	// cancellation until they're all done anyway.
	Transition(transitions []*sync.Change) ([]*sync.Entry, []*sync.Problem, bool, error)

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
