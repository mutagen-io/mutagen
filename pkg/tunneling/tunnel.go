package tunneling

import (
	"errors"
	"fmt"

	"github.com/mutagen-io/mutagen/pkg/identifier"
	"github.com/mutagen-io/mutagen/pkg/selection"
)

// EnsureValid ensures that TunnelHostCredentials' invariants are respected.
func (c *TunnelHostCredentials) EnsureValid() error {
	// Ensure that the parameters are non-nil.
	if c == nil {
		return errors.New("nil parameters")
	}

	// Ensure that the tunnel identifier is valid.
	if !identifier.IsValid(c.Identifier) {
		return errors.New("invalid tunnel identifier")
	}

	// Ensure that the tunnel version is supported.
	if !c.Version.Supported() {
		return errors.New("unknown or unsupported tunnel version")
	}

	// Ensure that the creation time is present.
	if c.CreationTime == nil {
		return errors.New("missing creation time")
	}

	// Ensure that the token is present.
	if c.Token == "" {
		return errors.New("empty token")
	}

	// Ensure that the secret has the correct length.
	if len(c.Secret) != c.Version.secretLength() {
		return errors.New("secret has incorrect length")
	}

	// Ensure that the configuration is valid.
	if err := c.Configuration.EnsureValid(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Success.
	return nil
}

// EnsureValid ensures that Tunnel's invariants are respected.
func (t *Tunnel) EnsureValid() error {
	// Ensure that the tunnel is non-nil.
	if t == nil {
		return errors.New("nil tunnel")
	}

	// Ensure that the tunnel identifier is valid.
	if !identifier.IsValid(t.Identifier) {
		return errors.New("invalid tunnel identifier")
	}

	// Ensure that the tunnel version is supported.
	if !t.Version.Supported() {
		return errors.New("unknown or unsupported tunnel version")
	}

	// Ensure that the creation time is present.
	if t.CreationTime == nil {
		return errors.New("missing creation time")
	}

	// Ensure that the token is present.
	if t.Token == "" {
		return errors.New("empty token")
	}

	// Ensure that the secret has the correct length.
	if len(t.Secret) != t.Version.secretLength() {
		return errors.New("secret has incorrect length")
	}

	// Ensure that the configuration is valid.
	if err := t.Configuration.EnsureValid(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Validate the tunnel name.
	if err := selection.EnsureNameValid(t.Name); err != nil {
		return fmt.Errorf("invalid tunnel name: %w", err)
	}

	// Ensure that labels are valid.
	for k, v := range t.Labels {
		if err := selection.EnsureLabelKeyValid(k); err != nil {
			return fmt.Errorf("invalid label key: %w", err)
		} else if err = selection.EnsureLabelValueValid(v); err != nil {
			return fmt.Errorf("invalid label value: %w", err)
		}
	}

	// Success.
	return nil
}
