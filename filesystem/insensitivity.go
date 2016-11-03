package filesystem

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/pkg/errors"
)

// insensitive checks if a filesystem is insensitive to a particular change in a
// path. For example, it can be used to check if two paths, say "UPPERlower" and
// "upperLOWER", refer to the same path. It uses a temporary file which it
// creates in a non-conflicting manner in root. If root is the empty string, the
// standard system temporary directory is used. The create string is used as a
// prefix for the temporary file name. It is then replaced with the check string
// and an attempt to access the file by that name is made. If the attempt fails,
// it means the filesystem is sensitive to the change. If the attempt succeeds,
// it means the filesystem is insensitive to the change. If an error is
// returned, the sensitivity value should not be used.
// TODO: Make this behave better for read-only filesystems.
func insensitive(root, create, check string) (bool, error) {
	// Create a temporary file using the creation name.
	file, err := ioutil.TempFile(root, create)
	if err != nil {
		return false, errors.Wrap(err, "unable to create temporary file")
	}

	// Grab the file name. This isn't read from the OS, it's the name computed
	// using the creation prefix, so it won't be changed in any way.
	name := file.Name()

	// Schedule the file for closure and removal.
	defer os.Remove(name)
	defer file.Close()

	// Perform the replacement and try to access the file by that name.
	if _, err = os.Stat(strings.Replace(name, create, check, -1)); err == nil {
		// We were able to access the file, which means the system is
		// insensitive to the replacement.
		return true, nil
	}

	// We weren't able to access the file by the alternate name, so the system
	// must be sensitive to the replacement.
	return false, nil
}
