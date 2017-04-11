package session

import (
	"hash"
	"io"
	"io/ioutil"
	"os"
	pathpkg "path"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/sync"
)

type stagingPathFinder struct {
	paths []string
}

func (f *stagingPathFinder) find(path string, entry *sync.Entry) error {
	// If the entry is non-existent, nothing needs to be staged.
	if entry == nil {
		return nil
	}

	// Handle based on type.
	if entry.Kind == sync.EntryKind_File {
		f.paths = append(f.paths, path)
	} else if entry.Kind == sync.EntryKind_Directory {
		for name, entry := range entry.Contents {
			if err := f.find(pathpkg.Join(path, name), entry); err != nil {
				return err
			}
		}
	} else {
		return errors.New("unknown entry type encountered")
	}

	// Success.
	return nil
}

func stagingPathsForChanges(changes []sync.Change) ([]string, error) {
	// Create a path finder.
	finder := &stagingPathFinder{}

	// Have it find paths for all the changes.
	for _, c := range changes {
		if err := finder.find(c.Path, c.New); err != nil {
			return nil, errors.Wrap(err, "unable to find staging paths")
		}
	}

	// Success.
	return finder.paths, nil
}

type stagingSink struct {
	session string
	alpha   bool
	// path is the path that is being staged. It is not the path to the storage
	// or the staging destination.
	path     string
	storage  *os.File
	digester hash.Hash
}

func (s *stagingSink) Write(data []byte) (int, error) {
	// Write to the underlying storage.
	n, err := s.storage.Write(data)

	// Write as much to the digester as we wrote to the underlying storage. This
	// can't fail.
	s.digester.Write(data[:n])

	// Done.
	return n, err
}

func (s *stagingSink) Close() error {
	// Close the underlying storage.
	if err := s.storage.Close(); err != nil {
		return errors.Wrap(err, "unable to close underlying storage")
	}

	// Compute the final digest.
	digest := s.digester.Sum(nil)

	// Compute where the file should be relocated.
	destination, err := pathForStaging(s.session, s.alpha, s.path, digest)
	if err != nil {
		os.Remove(s.storage.Name())
		return errors.Wrap(err, "unable to compute staging destination")
	}

	// Relocate the file to the destination.
	if err = os.Rename(s.storage.Name(), destination); err != nil {
		os.Remove(s.storage.Name())
		return errors.Wrap(err, "unable to relocate file")
	}

	// Success.
	return nil
}

// stagingCoordinator implements rsync.Sinker and sync.Provider.
type stagingCoordinator struct {
	session     string
	version     Version
	alpha       bool
	stagingRoot string
}

func newStagingCoordinator(session string, version Version, alpha bool) (*stagingCoordinator, error) {
	// Compute/create the staging root.
	stagingRoot, err := pathForStagingRoot(session, alpha)
	if err != nil {
		return nil, errors.Wrap(err, "unable to compute staging root")
	}

	// Success.
	return &stagingCoordinator{
		session:     session,
		version:     version,
		alpha:       alpha,
		stagingRoot: stagingRoot,
	}, nil
}

func (c *stagingCoordinator) wipe() error {
	// Remove the staging directory.
	if err := os.RemoveAll(c.stagingRoot); err != nil {
		return errors.Wrap(err, "unable to remove staging directory")
	}

	// Re-create the staging directory.
	if _, err := pathForStagingRoot(c.session, c.alpha); err != nil {
		return errors.Wrap(err, "unable to re-create staging directory")
	}

	// Success.
	return nil
}

func (c *stagingCoordinator) Sink(path string) (io.WriteCloser, error) {
	// Create a temporary storage file in the staging root.
	storage, err := ioutil.TempFile(c.stagingRoot, "staging")
	if err != nil {
		return nil, errors.Wrap(err, "unable to create temporary storage file")
	}

	// Success.
	return &stagingSink{
		session:  c.session,
		alpha:    c.alpha,
		path:     path,
		storage:  storage,
		digester: c.version.hasher(),
	}, nil
}

func (c *stagingCoordinator) Provide(path string, entry *sync.Entry) (string, error) {
	// Compute the expected location of the file.
	expectedLocation, err := pathForStaging(c.session, c.alpha, path, entry.Digest)
	if err != nil {
		return "", errors.Wrap(err, "unable to compute staging path")
	}

	// Ensure that it has the correct permissions.
	permissions := os.FileMode(0600)
	if entry.Executable {
		permissions = os.FileMode(0700)
	}
	if err = os.Chmod(expectedLocation, permissions); err != nil {
		return "", errors.Wrap(err, "unable to set file permissions")
	}

	// Success.
	return expectedLocation, nil
}
