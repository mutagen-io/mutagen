package session

import (
	"hash"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/sync"
)

const (
	// byteMax is the maximum value a byte can take.
	byteMax = 1<<8 - 1
)

type stagingSink struct {
	coordinator *stagingCoordinator
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
	destination, prefix, err := pathForStaging(s.coordinator.root, s.path, digest)
	if err != nil {
		os.Remove(s.storage.Name())
		return errors.Wrap(err, "unable to compute staging destination")
	}

	// Ensure the prefix directory exists.
	if err = s.coordinator.ensurePrefixExists(prefix); err != nil {
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

// stagingCoordinator coordinates the reception of files via rsync (by
// implementing rsync.Sinker) and the provision of those files to transitions
// (by implementing sync.Provider). It is not safe for concurrent access, and
// each stagingSink it produces should be closed before another is created.
type stagingCoordinator struct {
	version       Version
	root          string
	prefixCreated map[string]bool
}

func newStagingCoordinator(session string, version Version, alpha bool) (*stagingCoordinator, error) {
	// Compute the staging root.
	root, err := pathForStagingRoot(session, alpha)
	if err != nil {
		return nil, errors.Wrap(err, "unable to compute staging root")
	}

	// Success.
	return &stagingCoordinator{
		version: version,
		root:    root,
	}, nil
}

func (c *stagingCoordinator) prepare() error {
	// Ensure the staging root exists.
	if err := os.MkdirAll(c.root, 0700); err != nil {
		return errors.Wrap(err, "unable to create staging root")
	}

	// Create the prefix creation tracker.
	c.prefixCreated = make(map[string]bool, byteMax)

	// Success.
	return nil
}

func (c *stagingCoordinator) ensurePrefixExists(prefix string) error {
	// Check if we've already created that prefix.
	if c.prefixCreated[prefix] {
		return nil
	}

	// Otherwise create it and mark it as created.
	if err := os.MkdirAll(filepath.Join(c.root, prefix), 0700); err != nil {
		return err
	}
	c.prefixCreated[prefix] = true

	// Success.
	return nil
}

func (c *stagingCoordinator) wipe() error {
	// Zero-out the prefix creation tracker.
	c.prefixCreated = nil

	// Remove the staging root.
	if err := os.RemoveAll(c.root); err != nil {
		errors.Wrap(err, "unable to remove staging directory")
	}

	// Success.
	return nil
}

func (c *stagingCoordinator) Sink(path string) (io.WriteCloser, error) {
	// Create a temporary storage file in the staging root.
	storage, err := ioutil.TempFile(c.root, "staging")
	if err != nil {
		return nil, errors.Wrap(err, "unable to create temporary storage file")
	}

	// Success.
	return &stagingSink{
		coordinator: c,
		path:        path,
		storage:     storage,
		digester:    c.version.hasher(),
	}, nil
}

func (c *stagingCoordinator) Provide(path string, entry *sync.Entry) (string, error) {
	// Compute the expected location of the file.
	expectedLocation, _, err := pathForStaging(c.root, path, entry.Digest)
	if err != nil {
		return "", errors.Wrap(err, "unable to compute staging path")
	}

	// Ensure that it has the correct permissions. This will fail if the file
	// doens't exist.
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
