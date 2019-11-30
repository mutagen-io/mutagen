package tunneling

import (
	"errors"
	"fmt"

	"github.com/mutagen-io/mutagen/pkg/selection"
)

// ensureValid verifies that a CreationSpecification is valid.
func (s *CreationSpecification) ensureValid() error {
	// Ensure that the creation specification is non-nil.
	if s == nil {
		return errors.New("nil creation specification")
	}

	// Verify that the configuration is valid.
	if err := s.Configuration.EnsureValid(); err != nil {
		return fmt.Errorf("invalid tunnel configuration: %w", err)
	}

	// Verify that the name is valid.
	if err := selection.EnsureNameValid(s.Name); err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}

	// Verify that labels are valid.
	for k, v := range s.Labels {
		if err := selection.EnsureLabelKeyValid(k); err != nil {
			return fmt.Errorf("invalid label key: %w", err)
		} else if err = selection.EnsureLabelValueValid(v); err != nil {
			return fmt.Errorf("invalid label value: %w", err)
		}
	}

	// There's no need to validate the Paused field - either value is valid.

	// Success.
	return nil
}

// ensureValid verifies that a CreateRequest is valid.
func (r *CreateRequest) ensureValid(first bool) error {
	// Ensure that the request is non-nil.
	if r == nil {
		return errors.New("nil request")
	}

	// Handle validation based on whether or not this is the first request in
	// the stream.
	if first {
		// Verify that the creation specification is valid.
		if err := r.Specification.ensureValid(); err != nil {
			return err
		}

		// Verify that the response field is empty.
		if r.Response != "" {
			return errors.New("non-empty prompt response")
		}
	} else {
		// Verify that the creation specification is nil.
		if r.Specification != nil {
			return errors.New("creation specification present")
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
	// Ensure that the response is non-nil.
	if r == nil {
		return errors.New("nil response")
	}

	// Count the number of fields that are set.
	var fieldsSet uint
	if r.HostCredentials != nil {
		fieldsSet++
	}
	if r.Message != "" {
		fieldsSet++
	}
	if r.Prompt != "" {
		fieldsSet++
	}

	// Enforce that exactly one field is set.
	if fieldsSet != 1 {
		return errors.New("incorrect number of fields set")
	}

	// If the tunnel host credentials are set, validate them.
	if r.HostCredentials != nil {
		if err := r.HostCredentials.EnsureValid(); err != nil {
			return fmt.Errorf("invalid tunnel host credentials: %w", err)
		}
	}

	// Success.
	return nil
}

// ensureValid verifies that a ListRequest is valid.
func (r *ListRequest) ensureValid() error {
	// Ensure that the request is non-nil.
	if r == nil {
		return errors.New("nil request")
	}

	// Validate the tunnel specification.
	if err := r.Selection.EnsureValid(); err != nil {
		return fmt.Errorf("invalid tunnel specification: %w", err)
	}

	// There's no need to validate the state index - any value is valid.

	// Success.
	return nil
}

// ensureValid verifies that a ListResponse is valid.
func (r *ListResponse) EnsureValid() error {
	// Ensure that the response is non-nil.
	if r == nil {
		return errors.New("nil response")
	}

	// Ensure that all states are valid.
	for _, s := range r.TunnelStates {
		if err := s.EnsureValid(); err != nil {
			return fmt.Errorf("invalid tunnel state: %w", err)
		}
	}

	// Success.
	return nil
}

// ensureValid verifies that a PauseRequest is valid.
func (r *PauseRequest) ensureValid(first bool) error {
	// Ensure that the request is non-nil.
	if r == nil {
		return errors.New("nil request")
	}

	// Handle validation based on whether or not this is the first request in
	// the stream.
	if first {
		// Validate the tunnel selection specification.
		if err := r.Selection.EnsureValid(); err != nil {
			return fmt.Errorf("invalid tunnel selection specification: %w", err)
		}
	} else {
		// Ensure that no tunnel selection specification is present when
		// acknowledging messages.
		if r.Selection != nil {
			return errors.New("non-nil tunnel selection specification on message acknowledgement")
		}
	}

	// Success.
	return nil
}

// EnsureValid verifies that a PauseResponse is valid.
func (r *PauseResponse) EnsureValid() error {
	// Ensure that the response is non-nil.
	if r == nil {
		return errors.New("nil response")
	}

	// We can't really verify the message field. Even an empty value may be
	// valid.

	// Success.
	return nil
}

// ensureValid verifies that a ResumeRequest is valid.
func (r *ResumeRequest) ensureValid(first bool) error {
	// Ensure that the request is non-nil.
	if r == nil {
		return errors.New("nil request")
	}

	// Handle validation based on whether or not this is the first request in
	// the stream.
	if first {
		// Validate the tunnel selection specification.
		if err := r.Selection.EnsureValid(); err != nil {
			return fmt.Errorf("invalid tunnel selection specification: %w", err)
		}

		// Verify that the response field is empty.
		if r.Response != "" {
			return errors.New("non-empty prompt response")
		}
	} else {
		// Ensure that no tunnel selection specification is present when
		// acknowledging messages.
		if r.Selection != nil {
			return errors.New("non-nil tunnel selection specification on message acknowledgement")
		}

		// We can't really validate the response field, and an empty value may
		// be appropriate. It's up to the process performing the prompting to
		// decide.
	}

	// Success.
	return nil
}

// EnsureValid verifies that a ResumeResponse is valid.
func (r *ResumeResponse) EnsureValid() error {
	// Ensure that the response is non-nil.
	if r == nil {
		return errors.New("nil response")
	}

	// Count the number of fields that are set.
	var fieldsSet uint
	if r.Message != "" {
		fieldsSet++
	}
	if r.Prompt != "" {
		fieldsSet++
	}

	// Enforce that at most a single field is set. Unlike CreateResponse, we
	// allow neither to be set, which indicates completion. In CreateResponse,
	// this completion is indicated by the tunnel host credentials being set.
	if fieldsSet > 1 {
		return errors.New("multiple fields set")
	}

	// Success.
	return nil
}

// ensureValid verifies that a TerminateRequest is valid.
func (r *TerminateRequest) ensureValid(first bool) error {
	// Ensure that the request is non-nil.
	if r == nil {
		return errors.New("nil request")
	}

	// Handle validation based on whether or not this is the first request in
	// the stream.
	if first {
		// Validate the tunnel selection specification.
		if err := r.Selection.EnsureValid(); err != nil {
			return fmt.Errorf("invalid tunnel selection specification: %w", err)
		}
	} else {
		// Ensure that no tunnel selection specification is present when
		// acknowledging messages.
		if r.Selection != nil {
			return errors.New("non-nil tunnel selection specification on message acknowledgement")
		}

		// We can't really validate the response field, and an empty value may
		// be appropriate, especially if this is just a message acknowledgement.
	}

	// Success.
	return nil
}

// EnsureValid verifies that a TerminateResponse is valid.
func (r *TerminateResponse) EnsureValid() error {
	// Ensure that the response is non-nil.
	if r == nil {
		return errors.New("nil response")
	}

	// We can't really verify the message field. Even an empty value may be
	// valid.

	// Success.
	return nil
}
