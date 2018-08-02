package session

import (
	"github.com/havoc-io/mutagen/pkg/rsync"
	"github.com/havoc-io/mutagen/pkg/sync"
)

// initializeRequest encodes a request for endpoint initialization.
type initializeRequest struct {
	// Session is the session identifier.
	Session string
	// Version is the session version.
	Version Version
	// Root is the synchronization root path.
	Root string
	// Configuration is the session configuration.
	Configuration *Configuration
	// Alpha indicates whether or not the endpoint should behave as alpha (as
	// opposed to beta).
	Alpha bool
}

// initializeResponse encodes initialization results.
type initializeResponse struct {
	// Error is the error message (if any) resulting from initialization.
	Error string
}

// pollRequest encodes a request for one-shot polling.
type pollRequest struct{}

// pollCompletionRequest is paired with pollRequest and indicates a request for
// early polling completion or an acknowledgement of completion.
type pollCompletionRequest struct{}

// pollResponse indicates polling completion.
type pollResponse struct {
	// Error is the error message (if any) resulting from polling.
	Error string
}

// scanRequest encodes a request for a scan.
type scanRequest struct {
	// BaseSnapshotSignature is the rsync signature to use as the base for
	// differentially transmitting snapshots.
	BaseSnapshotSignature rsync.Signature
}

// scanResponse encodes the results of a scan.
type scanResponse struct {
	// SnapshotDelta are the operations need to reconstruct the snapshot against
	// the specified base.
	SnapshotDelta []rsync.Operation
	// PreservesExecutability indicates whether or not the scan root preserves
	// POSIX executability bits.
	PreservesExecutability bool
	// Error is the error message (if any) resulting from scanning.
	Error string
	// TryAgain indicates whether or not the error is ephermeral.
	TryAgain bool
}

// stageRequest encodes a request for staging.
type stageRequest struct {
	// Entries maps the paths that need to be staged to their target digests.
	Entries map[string][]byte
}

// stageRespone encodes the results of staging initialization.
type stageResponse struct {
	// Paths are the paths that need to be staged (relative to the
	// synchronization root).
	Paths []string
	// Signatures are the rsync signatures of the paths needing to be staged.
	Signatures []rsync.Signature
	// Error is the error message (if any) resulting from staging
	// initialization.
	Error string
}

// supplyRequest indicates a request for supplying files.
type supplyRequest struct {
	// Paths are the paths to provide (relative to the synchronization root).
	Paths []string
	// Signatures are the rsync signatures of the paths needing to be staged.
	Signatures []rsync.Signature
}

// transitionRequest encodes a request for transition application.
type transitionRequest struct {
	// Transitions are the transitions that need to be applied.
	Transitions []*sync.Change
}

// transitionResponse encodes the results of transitioning.
type transitionResponse struct {
	// Results are the resulting contents post-transition.
	// HACK: We have to use Archive to wrap our Entry results here because gob
	// won't encode a nil pointer in this slice, and the results of transitions
	// may very well be nil. We probably ought to transition to Protocol Buffers
	// for the remote endpoint protocol eventually, if not fully fledged gRPC,
	// but that's going to require converting all of the rsync types to Protocol
	// Buffers, which I'm not quite read to do.
	Results []*sync.Archive
	// Problems are any problems encountered during the transition operation.
	Problems []*sync.Problem
	// Error is the error message (if any) resulting from the remote transition
	// method. This will always be an empty string since transition doesn't
	// return errors from local endpoints, but to match the endpoint interface
	// (which allows for transition errors due to network failures with remote
	// endpoints), we include this field.
	// TODO: Should we just remove this field? Doing so would rely on knowledge
	// of localEndpoint's transition behavior.
	Error string
}

// endpointRequest is a sum type that can transmit any type of endpoint request.
// Only the sent request will be non-nil.
type endpointRequest struct {
	// Poll represents a poll request.
	Poll *pollRequest
	// Scan represents a scan request.
	Scan *scanRequest
	// Stage represents a stage request.
	Stage *stageRequest
	// Supply represents a supply request.
	Supply *supplyRequest
	// Transition represents a transition request.
	Transition *transitionRequest
}
