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
	// TODO: In Go 1.9, os.Executable will be fixed for OpenBSD (it will do what
	// osext does at the moment), so switch to that and remove the osext
	// dependency, licensing, and vendored code.
	if path, err := osext.Executable(); err != nil {
		panic(errors.Wrap(err, "unable to compute current executable's path"))
	} else {
		Current.ExecutablePath = path
	}

	// Compute the current executable's parent path.
	// TODO: In Go 1.9, switch to just taking the parent directory of what's
	// returned by os.Executable. That's exactly what osext does at the moment.
	if path, err := osext.ExecutableFolder(); err != nil {
		panic(errors.Wrap(err, "unable to compute current executable's parent path"))
	} else {
		Current.ExecutableParentPath = path
	}
}
