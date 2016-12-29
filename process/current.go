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
	// TODO: In Go 1.8, there's going to be an os.Executable function that will
	// serve this exact same purpose, so switch to that and remove the osext
	// dependency, licensing, and vendored code.
	if path, err := osext.Executable(); err != nil {
		panic(errors.Wrap(err, "unable to compute current executable's path"))
	} else {
		Current.ExecutablePath = path
	}

	// Compute the current executable's parent path.
	// TODO: In Go 1.8, switch to just taking the parent directory of what's
	// returned by os.Executable.
	if path, err := osext.ExecutableFolder(); err != nil {
		panic(errors.Wrap(err, "unable to compute current executable's parent path"))
	} else {
		Current.ExecutableParentPath = path
	}
}
