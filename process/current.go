package process

import (
	"github.com/kardianos/osext"

	"github.com/pkg/errors"
)

// Current represents the current process.
var Current struct {
	ExecutablePath       string
	ExecutableParentPath string
}

func init() {
	// Compute the current executable's path.
	if path, err := osext.Executable(); err != nil {
		panic(errors.Wrap(err, "unable to compute current executable's path"))
	} else {
		Current.ExecutablePath = path
	}

	// Compute the current executable's parent path.
	if path, err := osext.ExecutableFolder(); err != nil {
		panic(errors.Wrap(err, "unable to compute current executable's parent path"))
	} else {
		Current.ExecutableParentPath = path
	}
}
