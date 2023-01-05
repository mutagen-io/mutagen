package staging

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
)

// existsAndIsDirectory returns true if the target path exists, is readable, and
// is a directory, otherwise it returns false.
func existsAndIsDirectory(path string) bool {
	metadata, err := os.Lstat(path)
	return err == nil && metadata.IsDir()
}

// mkdirAllowExist is a wrapper around os.Mkdir that allows a directory to exist
// without also allowing the creation of intermediate directories (as is the
// case with os.MkdirAll). It isn't atomic, but it's fine for staging purposes.
func mkdirAllowExist(name string, perm os.FileMode) error {
	if err := os.Mkdir(name, perm); err == nil {
		return nil
	} else if os.IsExist(err) && existsAndIsDirectory(name) {
		return nil
	} else {
		return err
	}
}

// pathForStaging computes the staging path for the specified path/digest
// relative to the staging root. It also returns the prefix directory byte value
// and name, though it does not create the prefix directory.
func pathForStaging(root, path string, digest []byte) (string, byte, string, error) {
	// Ensure that the digest is non-empty. We need at least one byte for the
	// staging prefix to be valid, but beyond that we don't know what digest
	// length is in-use (or might be in use in the future).
	if len(digest) == 0 {
		return "", 0, "", errors.New("entry digest too short")
	}
	prefixByte := digest[0]

	// Convert the digest to hexadecimal encoding and extract the prefix.
	digestHex := hex.EncodeToString(digest)
	prefix := digestHex[:2]

	// Compute the hexadecimal encoded digest of the path name.
	pathDigest := sha1.Sum([]byte(path))
	pathDigestHex := hex.EncodeToString(pathDigest[:])

	// Compute the staging name.
	stagingName := pathDigestHex + "_" + digestHex

	// Success.
	return filepath.Join(root, prefix, stagingName), prefixByte, prefix, nil
}

// stagingSink is an io.WriteCloser designed to be returned by stager.
type stagingSink struct {
	// stager is the parent stager.
	stager *Stager
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
		return fmt.Errorf("unable to close underlying storage: %w", err)
	}

	// Compute the final digest.
	digest := s.digester.Sum(nil)

	// Compute where the file should be relocated.
	destination, prefixByte, prefix, err := pathForStaging(s.stager.root, s.path, digest)
	if err != nil {
		os.Remove(s.storage.Name())
		return fmt.Errorf("unable to compute staging destination: %w", err)
	}

	// Ensure the prefix directory exists.
	if err = s.stager.ensurePrefixExists(prefixByte, prefix); err != nil {
		os.Remove(s.storage.Name())
		return fmt.Errorf("unable to create prefix directory: %w", err)
	}

	// Relocate the file to the destination.
	if err = filesystem.Rename(nil, s.storage.Name(), nil, destination, true); err != nil {
		os.Remove(s.storage.Name())
		return fmt.Errorf("unable to relocate file: %w", err)
	}

	// Success.
	return nil
}

// Stager is an implementation of the local.Stager interface. It uses an
// ephemeral content-addressable store, allowing files to be staged in a
// load-balanced fashion in a temporary directory and then rapidly located by
// their digests. It is not safe for concurrent access, and each sink that it
// produces should be closed before any other method is invoked.
type Stager struct {
	// root is the staging root path.
	root string
	// hideRoot indicates whether or not the staging root should be marked as
	// hidden when created.
	hideRoot bool
	// digester is the hash function to use when processing files.
	digester hash.Hash
	// maximumFileSize is the maximum allowed size for a single staged file.
	maximumFileSize uint64
	// rootExists indicates whether or not the staging root currently exists.
	rootExists bool
	// prefixExists tracks whether or not individual prefix directories exist.
	// It may contain false negatives but will never contain false positives.
	prefixExists [256]bool
}

// NewStager creates a new stager.
func NewStager(root string, hideRoot bool, digester hash.Hash, maximumFileSize uint64) *Stager {
	return &Stager{
		root:            root,
		hideRoot:        hideRoot,
		digester:        digester,
		maximumFileSize: maximumFileSize,
		rootExists:      existsAndIsDirectory(root),
	}
}

// Prepare implements local.Stager.Prepare.
func (s *Stager) Prepare() error {
	// Create the staging root if it doesn't already exist.
	if !s.rootExists {
		// Attempt to create the root directory.
		if err := os.Mkdir(s.root, 0700); err != nil {
			return fmt.Errorf("unable to create staging root: %w", err)
		}

		// Mark the directory as hidden, if requested.
		if s.hideRoot {
			if err := filesystem.MarkHidden(s.root); err != nil {
				return fmt.Errorf("unable to make staging root as hidden: %w", err)
			}
		}

		// Update our tracking.
		s.rootExists = true
	}

	// Success.
	return nil
}

// ensurePrefixExists ensures that the specified prefix directory exists within
// the staging root, using a cache to avoid inefficient recreation.
func (s *Stager) ensurePrefixExists(prefixByte byte, prefix string) error {
	// Check if we've already created that prefix.
	if s.prefixExists[prefixByte] {
		return nil
	}

	// Otherwise create the prefix and record its creation. We allow prefixes to
	// exist already in order to support staging resumption.
	if err := mkdirAllowExist(filepath.Join(s.root, prefix), 0700); err != nil {
		return err
	}
	s.prefixExists[prefixByte] = true

	// Success.
	return nil
}

// Sink implements rsync.Sinker.Sink.
func (s *Stager) Sink(path string) (io.WriteCloser, error) {
	// Create a temporary storage file in the staging root.
	storage, err := os.CreateTemp(s.root, "staging")
	if err != nil {
		return nil, fmt.Errorf("unable to create temporary storage file: %w", err)
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

// Provide implements core.Provider.Provide.
func (s *Stager) Provide(path string, digest []byte) (string, error) {
	// If the root doesn't exist, then there's no way the file exists, and we
	// can simply return. This is an important optimization path for initial
	// synchronization of a large directories, where we don't want to perform a
	// huge number of os.Lstat calls that we know will fail.
	if !s.rootExists {
		return "", os.ErrNotExist
	}

	// Compute the expected location of the file.
	expectedLocation, _, _, err := pathForStaging(s.root, path, digest)
	if err != nil {
		return "", fmt.Errorf("unable to compute staging path: %w", err)
	}

	// Ensure that the path exists (i.e. that it staged successfully with the
	// expected contents (the digest of which are encoded in the location)).
	if _, err := os.Lstat(expectedLocation); err != nil {
		if os.IsNotExist(err) {
			return "", err
		}
		return "", fmt.Errorf("unable to query staged file metadata: %w", err)
	}

	// Success.
	return expectedLocation, nil
}

// Finalize implements local.Stager.Finalize.
func (s *Stager) Finalize() error {
	// Reset the prefix creation tracker.
	s.prefixExists = [256]bool{}

	// Reset root creation tracking.
	s.rootExists = false

	// Remove the staging root.
	if err := os.RemoveAll(s.root); err != nil {
		return fmt.Errorf("unable to remove staging directory: %w", err)
	}

	// Success.
	return nil
}
