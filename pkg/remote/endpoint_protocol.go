package remote

import (
	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/rsync"
	"github.com/havoc-io/mutagen/pkg/session"
	"github.com/havoc-io/mutagen/pkg/sync"
)

// initializeRequest encodes a request for endpoint initialization.
type initializeRequest struct {
	// Session is the session identifier.
	Session string
	// Version is the session version.
	Version session.Version
	// Root is the synchronization root path.
	Root string
	// Configuration is the session configuration.
	Configuration *session.Configuration
	// Alpha indicates whether or not the endpoint should behave as alpha (as
	// opposed to beta).
	Alpha bool
}

// ensureValid ensures that the initializeRequest's invariants are respected.
func (r *initializeRequest) ensureValid() error {
	// A nil initialize request is not valid.
	if r == nil {
		return errors.New("nil initialize request")
	}

	// Ensure that the session identifier is non-empty.
	if r.Session == "" {
		return errors.New("empty session identifier")
	}

	// Ensure that the session version is supported.
	if !r.Version.Supported() {
		return errors.New("unsupported session version")
	}

	// Ensure that the root path is non-empty.
	if r.Root == "" {
		return errors.New("empty root path")
	}

	// Ensure that the configuration is valid.
	if err := r.Configuration.EnsureValid(session.ConfigurationSourceSession); err != nil {
		return errors.Wrap(err, "invalid configuration")
	}

	// Success.
	return nil
}

// initializeResponse encodes initialization results.
type initializeResponse struct {
	// Error is the error message (if any) resulting from initialization.
	Error string
}

// ensureValid ensures that the initializeResponse's invariants are respected.
func (r *initializeResponse) ensureValid() error {
	// A nil initialize response is not valid.
	if r == nil {
		return errors.New("nil initialize response")
	}

	// Success.
	return nil
}

// pollRequest encodes a request for one-shot polling.
type pollRequest struct{}

// ensureValid ensures that the pollRequest's invariants are respected.
func (r *pollRequest) ensureValid() error {
	// A nil poll request is not valid.
	if r == nil {
		return errors.New("nil poll request")
	}

	// Success.
	return nil
}

// pollCompletionRequest is paired with pollRequest and indicates a request for
// early polling completion or an acknowledgement of completion.
type pollCompletionRequest struct{}

// ensureValid ensures that the pollCompletionRequest's invariants are respected.
func (r *pollCompletionRequest) ensureValid() error {
	// A nil poll completion request is not valid.
	if r == nil {
		return errors.New("nil poll completion request")
	}

	// Success.
	return nil
}

// pollResponse indicates polling completion.
type pollResponse struct {
	// Error is the error message (if any) resulting from polling.
	Error string
}

// ensureValid ensures that the pollResponse's invariants are respected.
func (r *pollResponse) ensureValid() error {
	// A nil poll response is not valid.
	if r == nil {
		return errors.New("nil poll response")
	}

	// Success.
	return nil
}

// scanRequest encodes a request for a scan.
type scanRequest struct {
	// BaseSnapshotSignature is the rsync signature to use as the base for
	// differentially transmitting snapshots.
	BaseSnapshotSignature *rsync.Signature
}

// ensureValid ensures that the scanRequest's invariants are respected.
func (r *scanRequest) ensureValid() error {
	// A nil scan request is not valid.
	if r == nil {
		return errors.New("nil scan request")
	}

	// Ensure that the base snapshot signature is valid.
	if err := r.BaseSnapshotSignature.EnsureValid(); err != nil {
		return errors.Wrap(err, "invalid base snapshot signature")
	}

	// Success.
	return nil
}

// scanResponse encodes the results of a scan.
type scanResponse struct {
	// SnapshotDelta are the operations need to reconstruct the snapshot against
	// the specified base.
	SnapshotDelta []*rsync.Operation
	// PreservesExecutability indicates whether or not the scan root preserves
	// POSIX executability bits.
	PreservesExecutability bool
	// Error is the error message (if any) resulting from scanning.
	Error string
	// TryAgain indicates whether or not the error is ephermeral.
	TryAgain bool
}

// ensureValid ensures that the scanResponse's invariants are respected.
func (r *scanResponse) ensureValid() error {
	// A nil scan response is not valid.
	if r == nil {
		return errors.New("nil scan response")
	}

	// Ensure that each snapshot delta operation is valid.
	for _, operation := range r.SnapshotDelta {
		if err := operation.EnsureValid(); err != nil {
			return errors.Wrap(err, "invalid snapshot delta operation")
		}
	}

	// If an error is set, make sure that certain other fields are not. This
	// isn't really an invariant that *needs* to be enforced, but it is a good
	// sanity check.
	if r.Error != "" {
		if len(r.SnapshotDelta) > 0 {
			return errors.New("non-empty snapshot delta present on error")
		} else if r.PreservesExecutability {
			return errors.New("executability preservation information present on error")
		}
	}

	// Success.
	return nil
}

// stageRequest encodes a request for staging.
type stageRequest struct {
	// Entries maps the paths that need to be staged to their target digests.
	Entries map[string][]byte
}

// ensureValid ensures that the stageRequest's invariants are respected.
func (r *stageRequest) ensureValid() error {
	// A nil stage request is not valid.
	if r == nil {
		return errors.New("nil stage request")
	}

	// Ensure that there are a non-zero number of entries. This isn't an
	// invariant that we really *need* to enforce, as our logic is capable of
	// handling it, but it's a useful check to make sure that the client is
	// avoiding transmission in these cases.
	if len(r.Entries) == 0 {
		return errors.New("no entries present")
	}

	// Success.
	return nil
}

// stageRespone encodes the results of staging initialization.
type stageResponse struct {
	// Paths are the paths that need to be staged (relative to the
	// synchronization root).
	Paths []string
	// Signatures are the rsync signatures of the paths needing to be staged.
	Signatures []*rsync.Signature
	// Error is the error message (if any) resulting from staging
	// initialization.
	Error string
}

// ensureValid ensures that stageResponse's invariants are respected.
func (r *stageResponse) ensureValid() error {
	// A nil stage response is not valid.
	if r == nil {
		return errors.New("nil stage response")
	}

	// Ensure that the number of paths matches the number of signatures.
	if len(r.Paths) != len(r.Signatures) {
		return errors.New("number of paths not equal to number of signatures")
	}

	// Verify that all signatures are valid.
	for _, signature := range r.Signatures {
		if err := signature.EnsureValid(); err != nil {
			return errors.Wrap(err, "invalid rsync signature")
		}
	}

	// Verify that paths and signatures are not present if there's an error.
	if r.Error != "" {
		if len(r.Paths) > 0 {
			return errors.New("paths/signatures present on error")
		}
	}

	// Success.
	return nil
}

// supplyRequest indicates a request for supplying files.
type supplyRequest struct {
	// Paths are the paths to provide (relative to the synchronization root).
	Paths []string
	// Signatures are the rsync signatures of the paths needing to be staged.
	Signatures []*rsync.Signature
}

// ensureValid ensures that supplyRequest's invariants are respected.
func (r *supplyRequest) ensureValid() error {
	// A nil supply request is not valid.
	if r == nil {
		return errors.New("nil supply request")
	}

	// Ensure that the number of paths matches the number of signatures.
	if len(r.Paths) != len(r.Signatures) {
		return errors.New("number of paths does not match number of signatures")
	}

	// Ensure that all signatures are valid.
	for _, s := range r.Signatures {
		if err := s.EnsureValid(); err != nil {
			return errors.Wrap(err, "invalid base signature detected")
		}
	}

	// Success.
	return nil
}

// transitionRequest encodes a request for transition application.
type transitionRequest struct {
	// Transitions are the transitions that need to be applied.
	Transitions []*sync.Change
}

// ensureValid ensures that transitionRequest's invariants are respected.
func (r *transitionRequest) ensureValid() error {
	// A nil transition request is not valid.
	if r == nil {
		return errors.New("nil transition request")
	}

	// Ensure that each change is valid.
	for _, change := range r.Transitions {
		if err := change.EnsureValid(); err != nil {
			return errors.Wrap(err, "invalid transition")
		}
	}

	// Success.
	return nil
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

// ensureValid ensures that transitionResponse's invariants are respected.
func (r *transitionResponse) ensureValid(expectedCount int) error {
	// A nil transition response is not valid.
	if r == nil {
		return errors.New("nil transition response")
	}

	// Ensure that the number of results matches the number expected.
	if len(r.Results) != expectedCount {
		return errors.New("unexpected number of results returned")
	}

	// Validate that each result is a valid entry specification.
	for _, result := range r.Results {
		if err := result.Root.EnsureValid(); err != nil {
			return errors.Wrap(err, "invalid result returned")
		}
	}

	// Validate that each problem is a valid problem specification.
	for _, problem := range r.Problems {
		if err := problem.EnsureValid(); err != nil {
			return errors.Wrap(err, "invalid problem returned")
		}
	}

	// Success.
	return nil
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

// ensureValid ensures that endpointRequest's invariants are respected.
func (r *endpointRequest) ensureValid() error {
	// A nil endpoint request is not valid.
	if r == nil {
		return errors.New("nil endpoint request")
	}

	// Ensure that exactly one field is set.
	set := 0
	if r.Poll != nil {
		set++
	}
	if r.Scan != nil {
		set++
	}
	if r.Stage != nil {
		set++
	}
	if r.Supply != nil {
		set++
	}
	if r.Transition != nil {
		set++
	}
	if set != 1 {
		return errors.New("invalid number of fields set")
	}

	// Success.
	return nil
}
