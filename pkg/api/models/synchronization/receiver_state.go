package synchronization

import (
	"github.com/mutagen-io/mutagen/pkg/synchronization/rsync"
)

// ReceiverState represents the status of an rsync receiver.
type ReceiverState struct {
	// Path is the path currently being received.
	Path string `json:"path"`
	// ReceivedSize is the number of bytes that have been received for the
	// current path from both block and data operations.
	ReceivedSize uint64 `json:"receivedSize"`
	// ExpectedSize is the number of bytes expected for the current path.
	ExpectedSize uint64 `json:"expectedSize"`
	// ReceivedFiles is the number of files that have already been received.
	ReceivedFiles uint64 `json:"receivedFiles"`
	// ExpectedFiles is the total number of files expected.
	ExpectedFiles uint64 `json:"expectedFiles"`
	// TotalReceivedSize is the total number of bytes that have been received
	// for all paths from both block and data operations.
	TotalReceivedSize uint64 `json:"totalReceivedSize"`
}

// newReceiverStateFromInternalReceiverState creates a new receiver state
// representation from an internal Protocol Buffers representation. The receiver
// state must be valid.
func newReceiverStateFromInternalReceiverState(state *rsync.ReceiverState) *ReceiverState {
	// If the state is nil, then return a nil state.
	if state == nil {
		return nil
	}

	// Perform conversion.
	return &ReceiverState{
		Path:              state.Path,
		ReceivedSize:      state.ReceivedSize,
		ExpectedSize:      state.ExpectedSize,
		ReceivedFiles:     state.ReceivedFiles,
		ExpectedFiles:     state.ExpectedFiles,
		TotalReceivedSize: state.TotalReceivedSize,
	}
}
