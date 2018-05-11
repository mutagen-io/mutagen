package session

import (
	"hash"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/sync"
)

const (
	// numberOfByteValues is the number of values a byte can take.
	numberOfByteValues = 1 << 8
)

type stagingSink struct {
	stager *stager
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
	destination, prefix, err := pathForStaging(s.stager.root, s.path, digest)
	if err != nil {
		os.Remove(s.storage.Name())
		return errors.Wrap(err, "unable to compute staging destination")
	}

	// Ensure the prefix directory exists.
	if err = s.stager.ensurePrefixExists(prefix); err != nil {
		os.Remove(s.storage.Name())
		return errors.Wrap(err, "unable to create prefix directory")
	}

	// Relocate the file to the destination.
	if err = os.Rename(s.storage.Name(), destination); err != nil {
		os.Remove(s.storage.Name())
		return errors.Wrap(err, "unable to relocate file")
	}

	// Success.
	return nil
}

// stager coordinates the reception of files via rsync (by implementing
// rsync.Sinker) and the provision of those files to transitions (by
// implementing sync.Provider). It is not safe for concurrent access, and each
// stagingSink it produces should be closed before another is created.
type stager struct {
	// version is the session version.
	version Version
	// root is the staging root.
	root string
	// rootCreated indicates whether or not the staging root has been created
	// by us since the last wipe.
	rootCreated bool
	// prefixCreated maps prefix names (e.g. "00" - "ff") to a boolean
	// indicating whether or not the prefix has been created by us since the
	// last wipe.
	prefixCreated map[string]bool
}

func newStager(session string, version Version, alpha bool) (*stager, error) {
	// Compute the staging root.
	root, err := pathForStagingRoot(session, alpha)
	if err != nil {
		return nil, errors.Wrap(err, "unable to compute staging root")
	}

	// Success.
	return &stager{
		version:       version,
		root:          root,
		prefixCreated: make(map[string]bool, numberOfByteValues),
	}, nil
}

func (s *stager) ensurePrefixExists(prefix string) error {
	// Check if we've already created that prefix.
	if s.prefixCreated[prefix] {
		return nil
	}

	// Otherwise create it and mark it as created. We can also mark the root as
	// created since it'll be an intermediate directory.
	if err := os.MkdirAll(filepath.Join(s.root, prefix), 0700); err != nil {
		return err
	}
	s.rootCreated = true
	s.prefixCreated[prefix] = true

	// Success.
	return nil
}

func (s *stager) wipe() error {
	// Reset the prefix creation tracker.
	s.prefixCreated = make(map[string]bool, numberOfByteValues)

	// Reset root creation tracking.
	s.rootCreated = false

	// Remove the staging root.
	if err := os.RemoveAll(s.root); err != nil {
		errors.Wrap(err, "unable to remove staging directory")
	}

	// Success.
	return nil
}

func (s *stager) Sink(path string) (io.WriteCloser, error) {
	// Create the staging root if we haven't already.
	if !s.rootCreated {
		if err := os.MkdirAll(s.root, 0700); err != nil {
			return nil, errors.Wrap(err, "unable to create staging root")
		}
		s.rootCreated = true
	}

	// Create a temporary storage file in the staging root.
	storage, err := ioutil.TempFile(s.root, "staging")
	if err != nil {
		return nil, errors.Wrap(err, "unable to create temporary storage file")
	}

	// Success.
	return &stagingSink{
		stager:   s,
		path:     path,
		storage:  storage,
		digester: s.version.hasher(),
	}, nil
}

func (s *stager) Provide(path string, entry *sync.Entry, baseMode os.FileMode) (string, error) {
	// Compute the expected location of the file.
	expectedLocation, _, err := pathForStaging(s.root, path, entry.Digest)
	if err != nil {
		return "", errors.Wrap(err, "unable to compute staging path")
	}

	// Compute the file mode.
	mode := baseMode
	if mode == 0 {
		mode = sync.ProviderBaseMode
	}
	if entry.Executable {
		mode |= sync.UserExecutablePermission
	} else {
		mode &^= sync.AnyExecutablePermission
	}

	// Ensure that it has the correct mode. This will fail if the file doesn't
	// exist.
	if err = os.Chmod(expectedLocation, mode); err != nil {
		return "", errors.Wrap(err, "unable to set file mode")
	}

	// Success.
	return expectedLocation, nil
}
