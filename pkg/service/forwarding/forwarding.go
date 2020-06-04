package forwarding

import (
	"errors"
	"fmt"

	"github.com/mutagen-io/mutagen/pkg/selection"
	"github.com/mutagen-io/mutagen/pkg/url"
)

// ensureValid verifies that a CreationSpecification is valid.
func (s *CreationSpecification) ensureValid() error {
	// A nil creation specification is not valid.
	if s == nil {
		return errors.New("nil creation specification")
	}

	// Verify that the source URL is valid and is a forwarding URL.
	if err := s.Source.EnsureValid(); err != nil {
		return fmt.Errorf("invalid source URL: %w", err)
	} else if s.Source.Kind != url.Kind_Forwarding {
		return errors.New("source URL is not a forwarding URL")
	}

	// Verify that the destination URL is valid and is a forwarding URL.
	if err := s.Destination.EnsureValid(); err != nil {
		return fmt.Errorf("invalid destination URL: %w", err)
	} else if s.Destination.Kind != url.Kind_Forwarding {
		return errors.New("destination URL is not a forwarding URL")
	}

	// Verify that the configuration is valid.
	if err := s.Configuration.EnsureValid(false); err != nil {
		return fmt.Errorf("invalid session configuration: %w", err)
	}

	// Verify that the source-specific configuration is valid.
	if err := s.ConfigurationSource.EnsureValid(true); err != nil {
		return fmt.Errorf("invalid source-specific configuration: %w", err)
	}

	// Verify that the destination-specific configuration is valid.
	if err := s.ConfigurationDestination.EnsureValid(true); err != nil {
		return fmt.Errorf("invalid destination-specific configuration: %w", err)
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
	// A nil create request is not valid.
	if r == nil {
		return errors.New("nil create request")
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
	// A nil create response is not valid.
	if r == nil {
		return errors.New("nil create response")
	}

	// Ensure that the session identifier is non-empty.
	if r.Session == "" {
		return errors.New("empty session identifier")
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

	// Ensure that the session selection is valid.
	if err := r.Selection.EnsureValid(); err != nil {
		return fmt.Errorf("invalid selection specification: %w", err)
	}

	// There's no need to validate the state index - any value is valid.

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
			return fmt.Errorf("invalid session state: %w", err)
		}
	}

	// Success.
	return nil
}

// ensureValid verifies that a PauseRequest is valid.
func (r *PauseRequest) ensureValid() error {
	// A nil pause request is not valid.
	if r == nil {
		return errors.New("nil pause request")
	}

	// Ensure that a prompter has been specified.
	if r.Prompter == "" {
		return errors.New("no prompter specified")
	}

	// Ensure that the session selection is valid.
	if err := r.Selection.EnsureValid(); err != nil {
		return fmt.Errorf("invalid selection specification: %w", err)
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

	// Success.
	return nil
}

// ensureValid verifies that a ResumeRequest is valid.
func (r *ResumeRequest) ensureValid() error {
	// A nil resume request is not valid.
	if r == nil {
		return errors.New("nil resume request")
	}

	// Ensure that a prompter has been specified.
	if r.Prompter == "" {
		return errors.New("no prompter specified")
	}

	// Ensure that the session selection is valid.
	if err := r.Selection.EnsureValid(); err != nil {
		return fmt.Errorf("invalid selection specification: %w", err)
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

	// Success.
	return nil
}

// ensureValid verifies that a TerminateRequest is valid.
func (r *TerminateRequest) ensureValid() error {
	// A nil terminate request is not valid.
	if r == nil {
		return errors.New("nil terminate request")
	}

	// Ensure that a prompter has been specified.
	if r.Prompter == "" {
		return errors.New("no prompter specified")
	}

	// Ensure that the session selection is valid.
	if err := r.Selection.EnsureValid(); err != nil {
		return fmt.Errorf("invalid selection specification: %w", err)
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

	// Success.
	return nil
}
