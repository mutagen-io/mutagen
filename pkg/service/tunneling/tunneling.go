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
func (r *CreateRequest) ensureValid() error {
	// Ensure that the request is non-nil.
	if r == nil {
		return errors.New("nil request")
	}

	// Ensure that a prompter has been specified.
	if r.Prompter == "" {
		return errors.New("no prompter specified")
	}

	// Ensure that the creation specification is valid.
	if err := r.Specification.ensureValid(); err != nil {
		return fmt.Errorf("invalid creation specification: %w", err)
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

	// Validate the host credentials.
	if err := r.HostCredentials.EnsureValid(); err != nil {
		return fmt.Errorf("invalid tunnel host credentials: %w", err)
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
		return fmt.Errorf("invalid selection specification: %w", err)
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
func (r *PauseRequest) ensureValid() error {
	// Ensure that the request is non-nil.
	if r == nil {
		return errors.New("nil request")
	}

	// Ensure that a prompter has been specified.
	if r.Prompter == "" {
		return errors.New("no prompter specified")
	}

	// Ensure that the tunnel selection is valid.
	if err := r.Selection.EnsureValid(); err != nil {
		return fmt.Errorf("invalid selection specification: %w", err)
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

	// Success.
	return nil
}

// ensureValid verifies that a ResumeRequest is valid.
func (r *ResumeRequest) ensureValid() error {
	// Ensure that the request is non-nil.
	if r == nil {
		return errors.New("nil request")
	}

	// Ensure that a prompter has been specified.
	if r.Prompter == "" {
		return errors.New("no prompter specified")
	}

	// Ensure that the tunnel selection is valid.
	if err := r.Selection.EnsureValid(); err != nil {
		return fmt.Errorf("invalid selection specification: %w", err)
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

	// Success.
	return nil
}

// ensureValid verifies that a TerminateRequest is valid.
func (r *TerminateRequest) ensureValid() error {
	// Ensure that the request is non-nil.
	if r == nil {
		return errors.New("nil request")
	}

	// Ensure that a prompter has been specified.
	if r.Prompter == "" {
		return errors.New("no prompter specified")
	}

	// Ensure that the tunnel selection is valid.
	if err := r.Selection.EnsureValid(); err != nil {
		return fmt.Errorf("invalid selection specification: %w", err)
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

	// Success.
	return nil
}
