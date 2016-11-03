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
