package local

import (
	"hash"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/filesystem"
)

const (
	// numberOfByteValues is the number of values a byte can take.
	numberOfByteValues = 1 << 8
)

// stagingSink is an io.WriteCloser designed to be returned by stager.
type stagingSink struct {
	// stager is the parent stager.
	stager *stager
	// path is the path that is being staged. It is not the path to the storage
	// or the staging destination.
	path string
	// storage is the temporary storage for the data.
	storage *os.File
	// digester is the hash of the data already written.
	digester hash.Hash
	// maximumSize is the maximum number of bytes allowed to be written to the
	// file.
	maximumSize uint64
	// currentSize is the number of bytes that have been written to the file.
	currentSize uint64
}

// Write writes data to the sink.
func (s *stagingSink) Write(data []byte) (int, error) {
	// Watch for size violations.
	if (s.maximumSize - s.currentSize) < uint64(len(data)) {
		return 0, errors.New("maximum file size reached")
	}

	// Write to the underlying storage.
	n, err := s.storage.Write(data)

	// Write as much to the digester as we wrote to the underlying storage. This
	// can't fail.
	s.digester.Write(data[:n])

	// Update the current size. We needn't worry about this overflowing, because
	// the check above is sufficient to ensure that this amount of data won't
	// overflow the maximum uint64 value.
	s.currentSize += uint64(n)

	// Done.
	return n, err
}

// Close closes the sink and moves the file into place.
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

// stager is an ephemeral content-addressable store implementation. It allows
// files to be staged in a load-balanced fashion in a temporary directory and
// then rapidly located by their digests. It implements both rsync.Sinker and
// sync.Provider. It is not safe for concurrent access, and each sink that it
// produces should be closed before any other method is invoked.
type stager struct {
	// root is the staging root path.
	root string
	// hideRoot indicates whether or not the staging root should be marked as
	// hidden.
	hideRoot bool
	// digester is the hash function to use when processing files.
	digester hash.Hash
	// maximumFileSize is the maximum allowed size for a single staged file.
	maximumFileSize uint64
	// rootCreated indicates whether or not the staging root has been created
	// by us since the last wipe.
	rootCreated bool
	// prefixCreated maps prefix names (e.g. "00" - "ff") to a boolean
	// indicating whether or not the prefix has been created by us since the
	// last wipe.
	prefixCreated map[string]bool
}

// newStager creates a new stager. Parent should be a common directory in which
// staging roots are created, and rootName should be the endpoint-unique name of
// the staging root to create/delete within the parent.
func newStager(root string, hideRoot bool, digester hash.Hash, maximumFileSize uint64) *stager {
	return &stager{
		root:            root,
		hideRoot:        hideRoot,
		digester:        digester,
		maximumFileSize: maximumFileSize,
		prefixCreated:   make(map[string]bool, numberOfByteValues),
	}
}

// ensurePrefixExists ensures that the specified prefix directory exists within
// the staging root, using a cache to avoid inefficient recreation.
func (s *stager) ensurePrefixExists(prefix string) error {
	// Check if we've already created that prefix.
	if s.prefixCreated[prefix] {
		return nil
	}

	// Otherwise create it and mark it as created. We can also mark the root as
	// created since it'll be an intermediate directory.
	if err := os.Mkdir(filepath.Join(s.root, prefix), 0700); err != nil {
		return err
	}
	s.rootCreated = true
	s.prefixCreated[prefix] = true

	// Success.
	return nil
}

// wipe removes the staging root.
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

// Sink implements the Sink method of rsync.Sinker.
func (s *stager) Sink(path string) (io.WriteCloser, error) {
	// Create the staging root if we haven't already.
	if !s.rootCreated {
		// Attempt to create the directory.
		if err := os.Mkdir(s.root, 0700); err != nil {
			return nil, errors.Wrap(err, "unable to create staging root")
		}

		// Mark the directory as hidden, if requested.
		if s.hideRoot {
			if err := filesystem.MarkHidden(s.root); err != nil {
				return nil, errors.Wrap(err, "unable to make staging root as hidden")
			}
		}

		// Update our creation tracking.
		s.rootCreated = true
	}

	// Create a temporary storage file in the staging root.
	storage, err := ioutil.TempFile(s.root, "staging")
	if err != nil {
		return nil, errors.Wrap(err, "unable to create temporary storage file")
	}

	// Reset the hash function state.
	s.digester.Reset()

	// Success.
	return &stagingSink{
		stager:      s,
		path:        path,
		storage:     storage,
		digester:    s.digester,
		maximumSize: s.maximumFileSize,
	}, nil
}

// Provide implements the Provide method of sync.Provider.
func (s *stager) Provide(path string, digest []byte) (string, error) {
	// Compute the expected location of the file.
	expectedLocation, _, err := pathForStaging(s.root, path, digest)
	if err != nil {
		return "", errors.Wrap(err, "unable to compute staging path")
	}

	// Ensure that the path exists (i.e. that it staged successfully with the
	// expected contents (the digest of which are encoded in the location)).
	if _, err := os.Lstat(expectedLocation); err != nil {
		if os.IsNotExist(err) {
			return "", err
		}
		return "", errors.Wrap(err, "unable to query staged file metadata")
	}

	// Success.
	return expectedLocation, nil
}
