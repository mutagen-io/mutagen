package filesystem

import (
	"strings"

	"github.com/pkg/errors"
)

// ErrUnsupportedOpenType indicates that the filesystem entry at the specified
// path is not supported as a traversal root.
var ErrUnsupportedOpenType = errors.New("unsupported open type")

// OpenDirectory is a convenience wrapper around Open that requires the result
// to be a directory.
func OpenDirectory(path string, allowSymbolicLinkLeaf bool) (*Directory, *Metadata, error) {
	if d, metadata, err := Open(path, allowSymbolicLinkLeaf); err != nil {
		return nil, nil, err
	} else if (metadata.Mode & ModeTypeMask) != ModeTypeDirectory {
		d.Close()
		return nil, nil, errors.New("path is not a directory")
	} else if directory, ok := d.(*Directory); !ok {
		d.Close()
		panic("invalid directory object returned from open operation")
	} else {
		return directory, metadata, nil
	}
}

// OpenFile is a convenience wrapper around Open that requires the result to be
// a file.
func OpenFile(path string, allowSymbolicLinkLeaf bool) (ReadableFile, *Metadata, error) {
	if f, metadata, err := Open(path, allowSymbolicLinkLeaf); err != nil {
		return nil, nil, err
	} else if (metadata.Mode & ModeTypeMask) != ModeTypeFile {
		f.Close()
		return nil, nil, errors.New("path is not a file")
	} else if file, ok := f.(ReadableFile); !ok {
		f.Close()
		panic("invalid file object returned from open operation")
	} else {
		return file, metadata, nil
	}
}

// Opener is a utility type that wraps a provided root path and provides file
// opening operations on paths relative to that root, guaranteeing that the open
// operations are performed in a race-free manner that can't escape the root via
// a path or symbolic link. It accomplishes this by maintaining an internal
// stack of Directory objects which provide this race-free opening property.
// This implementation means that the Opener operates "fast" if it is used to
// open paths in a sequence that mimics depth-first traversal ordering.
type Opener struct {
	// root is the root path for the opener.
	root string
	// rootDirectory is the Directory object corresponding to the root path. It
	// may be nil if the root directory hasn't been opened.
	rootDirectory *Directory
	// openParentNames is a list of parent directory names representing the
	// stack of currently open directories. It will be empty if rootDirectory is
	// nil.
	openParentNames []string
	// openParentDirectories is a list of parent Directory objects representing
	// the stack of currently open directories. Its length and contents
	// correspond to openParentNames, and likewise it will be empty if
	// rootDirectory is nil.
	openParentDirectories []*Directory
}

// NewOpener creates a new Opener for the specified root path.
func NewOpener(root string) *Opener {
	return &Opener{root: root}
}

// Open opens the file at the specified path (relative to the root). On all
// platforms, the path must be provided using a forward slash as the path
// separator, and path components must not be "." or "..". The path may be empty
// to open the root path itself (if it's a file). If any symbolic links or
// non-directory parent components are encountered, or if the target does not
// represent a file, this method will fail.
func (o *Opener) Open(path string) (ReadableFile, error) {
	// Handle the special case of a root path. We enforce that it must be a
	// file.
	if path == "" {
		// Verify that the root path hasn't already been opened as a directory.
		// This is primarily just a cheap sanity check. On POSIX systems, the
		// directory we hold open for the root could have been unlinked and
		// replaced with a file, and it's better to catch that here before
		// future Opener operations open files that aren't visible on the
		// filesystem or are somewhere else on the filesystem.
		if o.rootDirectory != nil {
			return nil, errors.New("root already opened as directory")
		}

		// Attempt to open the file.
		if file, _, err := OpenFile(o.root, false); err != nil {
			return nil, errors.Wrap(err, "unable to open root file")
		} else {
			return file, nil
		}
	}

	// Split the path and extract the parent components and leaf name.
	components := strings.Split(path, "/")
	parentComponents := components[:len(components)-1]
	leafName := components[len(components)-1]

	// If it's not already open, open the root directory.
	if o.rootDirectory == nil {
		if directory, _, err := OpenDirectory(o.root, false); err != nil {
			return nil, errors.Wrap(err, "unable to open root directory")
		} else {
			o.rootDirectory = directory
		}
	}

	// Identify the starting parent directory.
	parent := o.rootDirectory

	// Walk down parent components and open them.
	for c, component := range parentComponents {
		// See if we can satisfy the component requirement using our stacks. If
		// not, then truncate the stacks beyond this point.
		if c < len(o.openParentNames) {
			if o.openParentNames[c] == component {
				parent = o.openParentDirectories[c]
				continue
			} else {
				for i := c; i < len(o.openParentNames); i++ {
					// Attempt to close the directory.
					if err := o.openParentDirectories[i].Close(); err != nil {
						return nil, errors.Wrap(err, "unable to close previous parent directory")
					}

					// We nil-out successfully closed directories for two
					// reasons: first, to allow garbage collection, and second,
					// to work as sentinel values for the Close method.
					o.openParentNames[i] = ""
					o.openParentDirectories[i] = nil
				}
				o.openParentNames = o.openParentNames[:c]
				o.openParentDirectories = o.openParentDirectories[:c]
			}
		}

		// Open the directory ourselves and add it to the parent stacks.
		if directory, err := parent.OpenDirectory(component); err != nil {
			return nil, errors.Wrap(err, "unable to open parent directory")
		} else {
			parent = directory
			o.openParentNames = append(o.openParentNames, component)
			o.openParentDirectories = append(o.openParentDirectories, directory)
		}
	}

	// Open the leaf name within its parent directory.
	return parent.OpenFile(leafName)
}

// Close closes any open resources held by the opener. It should only be called
// once. Even on error, there is no benefit in calling it twice.
func (o *Opener) Close() error {
	// Track the first error to arise, if any.
	var firstErr error

	// Close the root directory, if open.
	if o.rootDirectory != nil {
		firstErr = o.rootDirectory.Close()
	}

	// Close open directories. If any are nil (which can happen on error
	// conditions in open when truncation doesn't complete successfully), then
	// just skip them.
	for _, directory := range o.openParentDirectories {
		if directory == nil {
			continue
		} else if err := directory.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	// Done.
	return firstErr
}
