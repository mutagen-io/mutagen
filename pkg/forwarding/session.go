package forwarding

import (
	"errors"
	"fmt"

	"github.com/mutagen-io/mutagen/pkg/identifier"
	"github.com/mutagen-io/mutagen/pkg/selection"
	"github.com/mutagen-io/mutagen/pkg/url"
)

// EnsureValid ensures that Session's invariants are respected.
func (s *Session) EnsureValid() error {
	// A nil session is not valid.
	if s == nil {
		return errors.New("nil session")
	}

	// Ensure that the session identifier is valid.
	if !identifier.IsValid(s.Identifier, true) {
		return errors.New("invalid session identifier")
	}

	// Ensure that the session version is supported.
	if !s.Version.Supported() {
		return errors.New("unknown or unsupported session version")
	}

	// Ensure that the creation time is present.
	if s.CreationTime == nil {
		return errors.New("missing creation time")
	}

	// Ensure that the source URL is valid and is a forwarding URL.
	if err := s.Source.EnsureValid(); err != nil {
		return fmt.Errorf("invalid source URL: %w", err)
	} else if s.Source.Kind != url.Kind_Forwarding {
		return errors.New("source URL is not a forwarding URL")
	}

	// Ensure that the destination URL is valid and is a forwarding URL.
	if err := s.Destination.EnsureValid(); err != nil {
		return fmt.Errorf("invalid destination URL: %w", err)
	} else if s.Destination.Kind != url.Kind_Forwarding {
		return errors.New("destination URL is not a forwarding URL")
	}

	// Ensure that the configuration is valid.
	if err := s.Configuration.EnsureValid(false); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Ensure that the source-specific configuration is valid.
	if err := s.ConfigurationSource.EnsureValid(true); err != nil {
		return fmt.Errorf("invalid source-specific configuration: %w", err)
	}

	// Ensure that the destination-specific configuration is valid.
	if err := s.ConfigurationDestination.EnsureValid(true); err != nil {
		return fmt.Errorf("invalid destination-specific configuration: %w", err)
	}

	// Validate the session name.
	if err := selection.EnsureNameValid(s.Name); err != nil {
		return fmt.Errorf("invalid session name: %w", err)
	}

	// Ensure that labels are valid.
	for k, v := range s.Labels {
		if err := selection.EnsureLabelKeyValid(k); err != nil {
			return fmt.Errorf("invalid label key: %w", err)
		} else if err = selection.EnsureLabelValueValid(v); err != nil {
			return fmt.Errorf("invalid label value: %w", err)
		}
	}

	// Success.
	return nil
}

// copy creates a static copy of the session, deep-copying any mutable members.
func (s *Session) copy() *Session {
	return &Session{
		Identifier:               s.Identifier,
		Version:                  s.Version,
		CreationTime:             s.CreationTime,
		CreatingVersionMajor:     s.CreatingVersionMajor,
		CreatingVersionMinor:     s.CreatingVersionMinor,
		CreatingVersionPatch:     s.CreatingVersionPatch,
		Source:                   s.Source,
		Destination:              s.Destination,
		Configuration:            s.Configuration,
		ConfigurationSource:      s.ConfigurationSource,
		ConfigurationDestination: s.ConfigurationDestination,
		Name:                     s.Name,
		Labels:                   s.Labels,
		Paused:                   s.Paused,
	}
}
