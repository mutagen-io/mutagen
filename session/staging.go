package session

import (
	"hash"
	"io"
	"io/ioutil"
	"os"
	"syscall"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/sync"
)

const (
	// byteMax is the maximum value a byte can take.
	byteMax = 1<<8 - 1
)

type stagingSink struct {
	root string
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
	destination, err := pathForStaging(s.root, s.path, digest)
	if err != nil {
		os.Remove(s.storage.Name())
		return errors.Wrap(err, "unable to compute staging destination")
	}

	// Relocate the file to the destination.
	// HACK: Just use syscall.Rename to avoid an os.Lstat call made by os.Rename
	// that can add significant overhead (~10% of total synchronization time) on
	// systems with slow stat implementations (e.g. macOS). We lose out on long
	// path fixes on Windows, but those shouldn't be necessary for paths inside
	// the staging directory (they are ~160 characters and up to 248 characters
	// is safe - see fixLongPath implementation in Go runtime on Windows). Even
	// on Linux this shaves about 2% of the time off.
	if err = syscall.Rename(s.storage.Name(), destination); err != nil {
		os.Remove(s.storage.Name())
		return errors.Wrap(err, "unable to relocate file")
	}

	// Success.
	return nil
}

// stagingCoordinator coordinates the reception of files via rsync (by
// implementing rsync.Sinker) and the provision of those files to transitions
// (by implementing sync.Provider).
type stagingCoordinator struct {
	version Version
	root    string
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
	// Create the staging root and all of its prefix directories. We keep this
	// functionality in with the other path functionality to keep all of that
	// logic together.
	return createStagingRootWithPrefixes(c.root)
}

func (c *stagingCoordinator) wipe() error {
	return errors.Wrap(os.RemoveAll(c.root), "unable to remove staging directory")
}

func (c *stagingCoordinator) Sink(path string) (io.WriteCloser, error) {
	// Create a temporary storage file in the staging root.
	storage, err := ioutil.TempFile(c.root, "staging")
	if err != nil {
		return nil, errors.Wrap(err, "unable to create temporary storage file")
	}

	// Success.
	return &stagingSink{
		root:     c.root,
		path:     path,
		storage:  storage,
		digester: c.version.hasher(),
	}, nil
}

func (c *stagingCoordinator) Provide(path string, entry *sync.Entry) (string, error) {
	// Compute the expected location of the file.
	expectedLocation, err := pathForStaging(c.root, path, entry.Digest)
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
