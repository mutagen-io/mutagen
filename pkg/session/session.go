package session

import (
	"github.com/pkg/errors"
)

// EnsureValid ensures that Session's invariants are respected.
func (s *Session) EnsureValid() error {
	// A nil session is not valid.
	if s == nil {
		return errors.New("nil session")
	}

	// Ensure that the session identifier is valid.
	// TODO: Should we validate with a UUID regex here?
	if s.Identifier == "" {
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

	// Ensure that the alpha URL is valid.
	if err := s.Alpha.EnsureValid(); err != nil {
		return errors.Wrap(err, "invalid alpha URL")
	}

	// Ensure that the beta URL is valid.
	if err := s.Beta.EnsureValid(); err != nil {
		return errors.Wrap(err, "invalid beta URL")
	}

	// Ensure that the configuration is valid.
	if err := s.Configuration.EnsureValid(false); err != nil {
		return errors.Wrap(err, "invalid configuration")
	}

	// Ensure that the alpha-specific configuration is valid.
	if err := s.ConfigurationAlpha.EnsureValid(true); err != nil {
		return errors.Wrap(err, "invalid alpha-specific configuration")
	}

	// Ensure that the beta-specific configuration is valid.
	if err := s.ConfigurationBeta.EnsureValid(true); err != nil {
		return errors.Wrap(err, "invalid beta-specific configuration")
	}

	// Ensure that labels are valid.
	for k, v := range s.Labels {
		if err := EnsureLabelKeyValid(k); err != nil {
			return errors.Wrap(err, "invalid label key")
		} else if err = EnsureLabelValueValid(v); err != nil {
			return errors.Wrap(err, "invalid label value")
		}
	}

	// Success.
	return nil
}
