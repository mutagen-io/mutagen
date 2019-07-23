package selection

import (
	"unicode"

	"github.com/pkg/errors"
)

// EnsureNameValid ensures that a name is valid for use as a session name. Empty
// names are treated as valid.
func EnsureNameValid(name string) error {
	// Loop over the string and ensure that its characters are allowed. At the
	// moment, the restrictions we apply here mirror those for Go identifiers.
	// We intentionally disallow dashes to avoid collisions with session UUID
	// identifiers. We might allow dashes (and maybe underscores) at some point
	// if there's a demand, at which point we'd have to ensure that the name
	// didn't match a UUID format (which might get a little expensive). The
	// current set of allowed characters also work as keys in YAML without
	// quoting.
	for i, r := range name {
		if unicode.IsLetter(r) {
			continue
		} else if i == 0 {
			return errors.New("name does not start with Unicode letter")
		} else if unicode.IsNumber(r) {
			continue
		}
		return errors.Errorf("invalid name character at index %d: '%c'", i, r)
	}

	// Disallow "defaults" as a session name since it is used as a special key
	// in YAML files.
	if name == "defaults" {
		return errors.New("\"defaults\" is disallowed as a session name")
	}

	// Success.
	return nil
}
