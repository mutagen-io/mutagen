package remote

import (
	"github.com/pkg/errors"
)

// ensureValid ensures that the InitializeRequest's invariants are respected.
func (r *InitializeRequest) ensureValid() error {
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
	if err := r.Configuration.EnsureValid(false); err != nil {
		return errors.Wrap(err, "invalid configuration")
	}

	// Success.
	return nil
}

// ensureValid ensures that the InitializeResponse's invariants are respected.
func (r *InitializeResponse) ensureValid() error {
	// A nil initialize response is not valid.
	if r == nil {
		return errors.New("nil initialize response")
	}

	// Success.
	return nil
}

// ensureValid ensures that the PollRequest's invariants are respected.
func (r *PollRequest) ensureValid() error {
	// A nil poll request is not valid.
	if r == nil {
		return errors.New("nil poll request")
	}

	// Success.
	return nil
}

// ensureValid ensures that the PollCompletionRequest's invariants are respected.
func (r *PollCompletionRequest) ensureValid() error {
	// A nil poll completion request is not valid.
	if r == nil {
		return errors.New("nil poll completion request")
	}

	// Success.
	return nil
}

// ensureValid ensures that the PollResponse's invariants are respected.
func (r *PollResponse) ensureValid() error {
	// A nil poll response is not valid.
	if r == nil {
		return errors.New("nil poll response")
	}

	// Success.
	return nil
}

// ensureValid ensures that the ScanRequest's invariants are respected.
func (r *ScanRequest) ensureValid() error {
	// A nil scan request is not valid.
	if r == nil {
		return errors.New("nil scan request")
	}

	// Ensure that the base snapshot signature is valid.
	if err := r.BaseSnapshotSignature.EnsureValid(); err != nil {
		return errors.Wrap(err, "invalid base snapshot signature")
	}

	// Full is correct regardless of value, so no validation is required.

	// Success.
	return nil
}

// ensureValid ensures that the ScanResponse's invariants are respected.
func (r *ScanResponse) ensureValid() error {
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

// ensureValid ensures that the StageRequest's invariants are respected.
func (r *StageRequest) ensureValid() error {
	// A nil stage request is not valid.
	if r == nil {
		return errors.New("nil stage request")
	}

	// Ensure that there are a non-zero number of paths. This isn't an invariant
	// that we really *need* to enforce, as our logic is capable of handling it,
	// but it's a useful check to make sure that the client is avoiding
	// transmission in these cases.
	if len(r.Paths) == 0 {
		return errors.New("no paths present")
	}

	// NOTE: We could perform an additional check that the specified paths are
	// unique, but this isn't quite so cheap, and it won't break anything if
	// they're not.

	// HACK: We don't verify that the paths are valid (and we'd have a hard time
	// doing so in any sense other than syntactically) because we use the
	// filesystem.Opener infrastructure to properly traverse the synchronization
	// root. It would also be expensive to verify the correctness of these paths
	// and it would be of little benefit. I'd class this as a hack because it's
	// sort of a layering violation (the message shouldn't know about the code
	// that uses it), but I'm willing to live with it because the message is so
	// tightly coupled to the endpoint implementation anyway.

	// Ensure that the number of digests matches the number of paths.
	if len(r.Digests) != len(r.Paths) {
		return errors.New("digest count does not match path count")
	}

	// NOTE: We could perform an additional check that the specified digests are
	// valid, but this isn't really necessary, and we'd have to handle varying
	// digest lengths.

	// Success.
	return nil
}

// ensureValid ensures that StageResponse's invariants are respected.
func (r *StageResponse) ensureValid() error {
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

// ensureValid ensures that SupplyRequest's invariants are respected.
func (r *SupplyRequest) ensureValid() error {
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

// ensureValid ensures that TransitionRequest's invariants are respected.
func (r *TransitionRequest) ensureValid() error {
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

// ensureValid ensures that TransitionResponse's invariants are respected.
func (r *TransitionResponse) ensureValid(expectedCount int) error {
	// A nil transition response is not valid.
	if r == nil {
		return errors.New("nil transition response")
	}

	// Ensure that the number of results matches the number expected.
	if len(r.Results) != expectedCount {
		return errors.New("unexpected number of results returned")
	}

	// Validate that each result is a valid archive specification.
	for _, result := range r.Results {
		if err := result.EnsureValid(); err != nil {
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

// ensureValid ensures that EndpointRequest's invariants are respected.
func (r *EndpointRequest) ensureValid() error {
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
