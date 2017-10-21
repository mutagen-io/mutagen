package process

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// Current represents the current process.
var Current struct {
	ExecutablePath       string
	ExecutableParentPath string
}

func init() {
	// Compute the current executable's path and parent path.
	if path, err := os.Executable(); err != nil {
		panic(errors.Wrap(err, "unable to compute current executable's path"))
	} else {
		Current.ExecutablePath = path
		Current.ExecutableParentPath = filepath.Dir(path)
	}
}
