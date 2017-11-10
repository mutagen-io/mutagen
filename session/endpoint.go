package session

import (
	"github.com/havoc-io/mutagen/rsync"
	"github.com/havoc-io/mutagen/sync"
)

// endpoint provides an interface to synchronization endpoints. It provides all
// primitives necessary to support synchronization. None of its methods are safe
// for concurrent invocation except close.
type endpoint interface {
	// poller returns a channel that will be populated when the endpoint is
	// marked as dirty. The channel has exactly one element of internal
	// buffering, so events will never be missed, redundant events won't be
	// stored, and no internal code will block waiting for a receive on the
	// channel. The channel will be closed when close is invoked. It will also
	// be closed if any internal watch error occurs, so any polling code should
	// also watch for that case.
	poller() chan struct{}

	// scan performs a scan of the endpoint's synchronization root. It requires
	// the ancestor to be passed in for executability propagation and for
	// optimized transfers if the endpoint is remote. It returns the scan
	// result, a bit indicating whether or not to re-try the scan, and any error
	// that occurred while trying to create the scan. Only one of these values
	// will be non-nil/false. If all are nil, it indicates that the
	// synchronization root doesn't exist on the endpoint, but that the scan
	// otherwise completed successfully.
	scan(ancestor *sync.Entry) (*sync.Entry, bool, error)

	// stage performs staging on the endpoint. It accepts a list of file paths
	// and file entries for those paths. It will filter the list based on what
	// it already has staged from previously interrupted stagings, and then
	// return a list of paths, their signatures, and a receiver to receive them.
	// The returned receiver must be finalized (i.e. transmitted to) before
	// subsequent methods can be invoked on the endpoint. If the receiver fails,
	// the endpoint should be considered contaminated and not used (though close
	// can and should still be invoked).
	stage(paths []string, entries []*sync.Entry) ([]string, []rsync.Signature, rsync.Receiver, error)

	// supply transmits files in a streaming fashion using the rsync algorithm
	// to the specified receiver.
	supply(paths []string, signatures []rsync.Signature, receiver rsync.Receiver) error

	// transition performs the specified transitions on the endpoint. It returns
	// a list of successfully applied changes and a list of problems that
	// occured while applying transitions.
	// TODO: Should we consider pre-emptability for transition? It could
	// probably be done by just checking for cancellation during each transition
	// path and reporting "cancelled" for problems arising after that, but
	// usually the long-blocking transitions are going to be the ones where
	// we're creating the root with a huge number of files and wouldn't catch
	// cancellation until they're all done anyway.
	transition(transitions []sync.Change) ([]sync.Change, []sync.Problem, error)

	// close terminates any resources associated with the endpoint. For local
	// endpoints, close will not preempt calls, but for remote endpoints it will
	// because it closes the underlying connection to the endpoint (actually, it
	// terminates that connection). close can safely be called concurrently with
	// other methods, though it's only recommended when you don't want the
	// possibility of preempting the method (e.g. in transition) or you know
	// that the operation can continue and terminate on its own (e.g. in scan).
	// close should only be invoked once.
	close() error
}
