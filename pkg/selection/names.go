package selection

import (
	"unicode"

	"github.com/pkg/errors"

	"github.com/google/uuid"
)

// EnsureNameValid ensures that a name is valid for use as a session name. Empty
// names are treated as valid.
func EnsureNameValid(name string) error {
	// Loop over the string and ensure that its characters are allowed. We allow
	// letters, numbers, and dashses, but we require that the identifier starts
	// with a letter. If a name contains a dash, then we enforce that it isn't a
	// UUID to avoid collisions with session identifiers.
	var containsDash bool
	for i, r := range name {
		if unicode.IsLetter(r) {
			continue
		} else if i == 0 {
			return errors.New("name does not start with Unicode letter")
		} else if unicode.IsNumber(r) {
			continue
		} else if r == '-' {
			containsDash = true
			continue
		}
		return errors.Errorf("invalid name character at index %d: '%c'", i, r)
	}

	// If the session name contains a dash, then ensure that it isn't a UUID.
	if containsDash {
		if _, err := uuid.Parse(name); err == nil {
			return errors.New("name must not be a UUID")
		}
	}

	// Disallow "defaults" as a session name since it is used as a special key
	// in YAML files.
	if name == "defaults" {
		return errors.New("\"defaults\" is disallowed as a session name")
	}

	// Success.
	return nil
}
