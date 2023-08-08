package ignore

import (
	"github.com/mutagen-io/mutagen/pkg/synchronization/core/fastpath"
)

// vcsDirectoryNames maps directory names to a boolean indicating whether or not
// they represent a VCS directory.
var vcsDirectoryNames = map[string]bool{
	".git":   true,
	".svn":   true,
	".hg":    true,
	".bzr":   true,
	"_darcs": true,
}

// vcsIgnorer is a wrapper Ignorer that provides VCS ignoring behavior.
type vcsIgnorer struct {
	// ignorer is the underlying ignorer.
	ignorer Ignorer
}

// Ignore implements Ignorer.Ignore.
func (i *vcsIgnorer) Ignore(path string, directory bool) (IgnoreStatus, bool) {
	// Watch for and ignore any VCS directories.
	if directory && vcsDirectoryNames[fastpath.Base(path)] {
		return IgnoreStatusIgnored, false
	}

	// Dispatch all other requests to the underlying ignorer.
	return i.ignorer.Ignore(path, directory)
}

// IgnoreVCS wraps an ignorer, modifying it to ignore VCS directories.
func IgnoreVCS(ignorer Ignorer) Ignorer {
	return &vcsIgnorer{ignorer}
}
