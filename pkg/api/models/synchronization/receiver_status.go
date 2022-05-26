package synchronization

import (
	"github.com/mutagen-io/mutagen/pkg/synchronization/rsync"
)

// ReceiverStatus represents the status of an rsync receiver.
type ReceiverStatus struct {
	// Path is the path currently being received.
	Path string `json:"path"`
	// Received is the number of paths that have already been received.
	Received uint64 `json:"received"`
	// Total is the total number of paths expected.
	Total uint64 `json:"total"`
}

// newReceiverStatusFromInternalReceiverStatus creates a new receiver status
// representation from an internal Protocol Buffers representation. The receiver
// status must be valid.
func newReceiverStatusFromInternalReceiverStatus(status *rsync.ReceiverStatus) *ReceiverStatus {
	// If the status is nil, then return a nil status.
	if status == nil {
		return nil
	}

	// Perform conversion.
	return &ReceiverStatus{
		Path:     status.Path,
		Received: status.Received,
		Total:    status.Total,
	}
}
