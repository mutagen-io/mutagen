// +build !windows

package filesystem

import (
	"os"

	"github.com/pkg/errors"
)

// mustComputeHomeDirectory computes the user's home directory and panics on
// failure. Since we're generally running without cgo on POSIX systems (other
// than macOS, from which we do our cross compiling and hence have cgo support),
// it's best to avoid the os/user package. It has a pure-Go implementation, but
// this implementation replies on the USER environment variable being set on
// POSIX systems, which isn't always the case, and which isn't necessary for our
// use case. Additionally, it only uses the HOME environment variable to grab
// the home directory anyway, which we can do just as easily.
func mustComputeHomeDirectory() string {
	// Grab the home directory from the environment and ensure that it's
	// non-empty.
	home, ok := os.LookupEnv("HOME")
	if !ok {
		panic(errors.New("HOME environment variable not present"))
	} else if home == "" {
		panic(errors.New("HOME environment variable empty"))
	}

	// Success.
	return home
}
