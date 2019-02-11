package session

import (
	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/session"
)

// ensureValid verifies that a CreateRequest is valid.
func (r *CreateRequest) ensureValid(first bool) error {
	// A nil create request is not valid.
	if r == nil {
		return errors.New("nil create request")
	}

	// Handle validation based on whether or not this is the first request in
	// the stream.
	if first {
		// Verify that the alpha URL is valid.
		if err := r.Alpha.EnsureValid(); err != nil {
			return errors.Wrap(err, "invalid alpha URL")
		}

		// Verify that the beta URL is valid.
		if err := r.Beta.EnsureValid(); err != nil {
			return errors.Wrap(err, "invalid beta URL")
		}

		// Verify that the configuration is valid.
		if err := r.Configuration.EnsureValid(session.ConfigurationSourceTypeCreate); err != nil {
			return errors.Wrap(err, "invalid session configuration")
		}

		// Verify that the alpha-specific configuration is valid.
		if err := r.ConfigurationAlpha.EnsureValid(session.ConfigurationSourceTypeCreateEndpointSpecific); err != nil {
			return errors.Wrap(err, "invalid alpha-specific configuration")
		}

		// Verify that the beta-specific configuration is valid.
		if err := r.ConfigurationBeta.EnsureValid(session.ConfigurationSourceTypeCreateEndpointSpecific); err != nil {
			return errors.Wrap(err, "invalid beta-specific configuration")
		}

		// Verify that the response field is empty.
		if r.Response != "" {
			return errors.New("non-empty response")
		}
	} else {
		// Verify that the alpha URL is nil.
		if r.Alpha != nil {
			return errors.New("alpha URL present")
		}

		// Verify that the beta URL is nil.
		if r.Beta != nil {
			return errors.New("beta URL present")
		}

		// Verify that the configuration is nil.
		if r.Configuration != nil {
			return errors.New("configuration present")
		}

		// Verify that the alpha-specific configuration is nil.
		if r.ConfigurationAlpha != nil {
			return errors.New("alpha-specific configuration present")
		}

		// Verify that the beta-specific configuration is nil.
		if r.ConfigurationBeta != nil {
			return errors.New("beta-specific configuration present")
		}

		// We can't really validate the response field, and an empty value may
		// be appropriate. It's up to the process performing the prompting to
		// decide.
	}

	// Success.
	return nil
}

// EnsureValid verifies that a CreateResponse is valid.
func (r *CreateResponse) EnsureValid() error {
	// A nil create response is not valid.
	if r == nil {
		return errors.New("nil create response")
	}

	// Ensure that exactly one field is set.
	var fieldsSet int
	if r.Session != "" {
		fieldsSet++
	}
	if r.Message != "" {
		fieldsSet++
	}
	if r.Prompt != "" {
		fieldsSet++
	}
	if fieldsSet != 1 {
		return errors.New("incorrect number of fields set")
	}

	// Success.
	return nil
}

// ensureValid verifies that a ListRequest is valid.
func (r *ListRequest) ensureValid() error {
	// A nil list request is not valid.
	if r == nil {
		return errors.New("nil list request")
	}

	// We can't really verify specifications, but any values are valid, even if
	// they might not be correct.

	// Success.
	return nil
}

// ensureValid verifies that a ListResponse is valid.
func (r *ListResponse) EnsureValid() error {
	// A nil list response is not valid.
	if r == nil {
		return errors.New("nil list response")
	}

	// Ensure that all states are valid.
	for _, s := range r.SessionStates {
		if err := s.EnsureValid(); err != nil {
			return errors.Wrap(err, "invalid session state")
		}
	}

	// Success.
	return nil
}

// ensureValid verifies that a FlushRequest is valid.
func (r *FlushRequest) ensureValid(first bool) error {
	// A nil flush request is not valid.
	if r == nil {
		return errors.New("nil flush request")
	}

	// Handle validation based on whether or not this is the first request in
	// the stream.
	if first {
		// We can't really verify specifications, but any values are valid, even
		// if they might not be correct.
	} else {
		// Ensure that specifications are empty when acknowledging messages.
		if r.Specifications != nil {
			return errors.New("non-empty specifications on message acknowledgement")
		}
	}

	// Success.
	return nil
}

// EnsureValid verifies that a FlushResponse is valid.
func (r *FlushResponse) EnsureValid() error {
	// A nil flush response is not valid.
	if r == nil {
		return errors.New("nil flush response")
	}

	// We can't really verify the message field. Even an empty value may be
	// valid.

	// Success.
	return nil
}

// ensureValid verifies that a PauseRequest is valid.
func (r *PauseRequest) ensureValid(first bool) error {
	// A nil pause request is not valid.
	if r == nil {
		return errors.New("nil pause request")
	}

	// Handle validation based on whether or not this is the first request in
	// the stream.
	if first {
		// We can't really verify specifications, but any values are valid, even
		// if they might not be correct.
	} else {
		// Ensure that specifications are empty when acknowledging messages.
		if r.Specifications != nil {
			return errors.New("non-empty specifications on message acknowledgement")
		}
	}

	// Success.
	return nil
}

// EnsureValid verifies that a PauseResponse is valid.
func (r *PauseResponse) EnsureValid() error {
	// A nil pause response is not valid.
	if r == nil {
		return errors.New("nil pause response")
	}

	// We can't really verify the message field. Even an empty value may be
	// valid.

	// Success.
	return nil
}

// ensureValid verifies that a ResumeRequest is valid.
func (r *ResumeRequest) ensureValid(first bool) error {
	// A nil resume request is not valid.
	if r == nil {
		return errors.New("nil resume request")
	}

	// Handle validation based on whether or not this is the first request in
	// the stream.
	if first {
		// We can't really verify specifications, but any values are valid, even
		// if they might not be correct.
	} else {
		// Ensure that specifications are empty when acknowledging messages.
		if r.Specifications != nil {
			return errors.New("non-empty specifications on message acknowledgement")
		}
	}

	// Success.
	return nil
}

// EnsureValid verifies that a ResumeResponse is valid.
func (r *ResumeResponse) EnsureValid() error {
	// A nil resume response is not valid.
	if r == nil {
		return errors.New("nil resume response")
	}

	// Ensure that at most a single field is set. Unlike CreateResponse, we
	// allow neither to be set, which indicates completion. In CreateResponse,
	// this completion is indicated by the session identifier being set.
	var fieldsSet int
	if r.Message != "" {
		fieldsSet++
	}
	if r.Prompt != "" {
		fieldsSet++
	}
	if fieldsSet > 1 {
		return errors.New("multiple fields set")
	}

	// Success.
	return nil
}

// ensureValid verifies that a TerminateRequest is valid.
func (r *TerminateRequest) ensureValid(first bool) error {
	// A nil terminate request is not valid.
	if r == nil {
		return errors.New("nil terminate request")
	}

	// Handle validation based on whether or not this is the first request in
	// the stream.
	if first {
		// We can't really verify specifications, but any values are valid, even
		// if they might not be correct.
	} else {
		// Ensure that specifications are empty when responding.
		if r.Specifications != nil {
			return errors.New("non-empty specifications on message acknowledgement")
		}

		// We can't really validate the response field, and an empty value may
		// be appropriate, especially if this is just a message acknowledgement.
	}

	// Success.
	return nil
}

// EnsureValid verifies that a TerminateResponse is valid.
func (r *TerminateResponse) EnsureValid() error {
	// A nil terminate response is not valid.
	if r == nil {
		return errors.New("nil terminate response")
	}

	// We can't really verify the message field. Even an empty value may be
	// valid.

	// Success.
	return nil
}
