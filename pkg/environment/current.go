package environment

import (
	"os"

	"github.com/pkg/errors"
)

var Current map[string]string

func init() {
	// Convert the standard environment variables into something more sensible.
	if current, err := Parse(os.Environ()); err != nil {
		panic(errors.Wrap(err, "unable to parse environment"))
	} else {
		Current = current
	}
}

func CopyCurrent() map[string]string {
	// Create a new environment map.
	result := make(map[string]string, len(Current))

	// Populate it.
	for k, v := range Current {
		result[k] = v
	}

	// Done.
	return result
}
