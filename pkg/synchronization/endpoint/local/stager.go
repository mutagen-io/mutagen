package local

import (
	"context"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"sync"

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

// concurrentHash implements a concurrent version of hash.Hash. While it
// operates concurrently, it is not safe for concurrent usage.
type concurrentHash struct {
	// cancel terminates the hashing Goroutine.
	cancel context.CancelFunc
	// done tracks termination of the hashing Goroutine.
	done sync.WaitGroup
	// resetRequests is used to trigger a reset.
	resetRequests chan struct{}
	// writeRequests is used to trigger a write.
	writeRequests chan []byte
	// writeResponses is used to wait for write completion.
	writeResponses chan struct{}
	// sumRequests is used to trigger a sum operation.
	sumRequests chan struct{}
	// sumDone is used to wait for a sum operation to complete.
	sumResponses chan []byte
}

// newConcurrentHash creates a new concurrent hash.
func newConcurrentHash(hasher hash.Hash) *concurrentHash {
	// Create a cancellable context for the hashing Goroutine.
	ctx, cancel := context.WithCancel(context.Background())

	// Create the result.
	result := &concurrentHash{
		cancel:         cancel,
		resetRequests:  make(chan struct{}),
		writeRequests:  make(chan []byte),
		writeResponses: make(chan struct{}),
		sumRequests:    make(chan struct{}),
		sumResponses:   make(chan []byte),
	}

	// Track hashing Goroutine completion.
	result.done.Add(1)

	// Start the hashing Goroutine.
	go result.run(ctx, hasher)

	// Done.
	return result
}

// shutdown terminates the run loop for the concurrent hash and awaits its
// completion.
func (h *concurrentHash) shutdown() {
	h.cancel()
	h.done.Wait()
}

// run is the run loop that performs asynchronous digests.
func (h *concurrentHash) run(ctx context.Context, hasher hash.Hash) {
	for {
		select {
		case <-ctx.Done():
			h.done.Done()
			return
		case <-h.resetRequests:
			hasher.Reset()
		case data := <- h.writeRequests:
			hasher.Write(data)
			h.writeResponses <- struct{}{}
		case <-h.sumRequests:
			h.sumResponses <- hasher.Sum(nil)
		}
	}
}

// reset triggers a resset operation on the hash.
func (h *concurrentHash) reset() {
	h.resetRequests <- struct{}{}
}

// write starts an asynchronous digest operation. It must be paired with a
// corresponding call to writeWait. The data must not be modified between the
// two calls.
func (h *concurrentHash) write(data []byte) {
	h.writeRequests <- data
}

// writeWait waits for completion of a write operation.
func (h *concurrentHash) writeWait() {
	<-h.writeResponses
}

// sum performs a sum operation.
func (h *concurrentHash) sum() []byte {
	h.sumRequests <- struct{}{}
	return <-h.sumResponses
}

// stagingSink is an io.WriteCloser designed to be returned by stager.
type stagingSink struct {
	// stager is the parent stager.
	stager *stager
	// path is the path that is being staged. It is not the path to the storage
	// or the staging destination.
	path string
	// storage is the temporary storage for the data.
	storage *os.File
	// maximumSize is the maximum number of bytes allowed to be written to the
	// file.
	maximumSize uint64
	// currentSize is the number of bytes that have been written to the file.
	currentSize uint64
	// previousWriteError is any previous write error that occurred.
	previousWriteError error
}

// Write writes data to the sink.
func (s *stagingSink) Write(data []byte) (int, error) {
	// If a previous write error occurred, then don't continue writing, because
	// the digest and the file may now differ in terms of processed content.
	if s.previousWriteError != nil {
		return 0, fmt.Errorf("previous write error: %w", s.previousWriteError)
	}

	// Watch for size violations.
	if (s.maximumSize - s.currentSize) < uint64(len(data)) {
		return 0, errors.New("maximum file size reached")
	}

	// Starting digesting the data asynchronously.
	s.stager.digester.write(data)

	// Write to the underlying storage.
	n, err := s.storage.Write(data)

	// Update the current size. We needn't worry about this overflowing, because
	// the check above is sufficient to ensure that this amount of data won't
	// overflow the maximum uint64 value.
	s.currentSize += uint64(n)

	// Record any error that occurred, because the digester and the file now
	// likely have different data written to them and we can't allow the staging
	// operation to continue.
	s.previousWriteError = err

	// Wait for the asynchronous digest operation to complete.
	s.stager.digester.writeWait()

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
	digest := s.stager.digester.sum()

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

// stager is an ephemeral content-addressable store implementation. It allows
// files to be staged in a load-balanced fashion in a temporary directory and
// then rapidly located by their digests. It implements both rsync.Sinker and
// sync.Provider. It is not safe for concurrent access, and each sink that it
// produces should be closed before any other method is invoked.
type stager struct {
	// root is the staging root path.
	root string
	// hideRoot indicates whether or not the staging root should be marked as
	// hidden when created.
	hideRoot bool
	// digester hashes incoming file content.
	digester *concurrentHash
	// maximumFileSize is the maximum allowed size for a single staged file.
	maximumFileSize uint64
	// rootExists indicates whether or not the staging root currently exists.
	rootExists bool
	// prefixExists tracks whether or not individual prefix directories exist.
	prefixExists [256]bool
}

// newStager creates a new stager.
func newStager(root string, hideRoot bool, digester hash.Hash, maximumFileSize uint64) *stager {
	return &stager{
		root:            root,
		hideRoot:        hideRoot,
		digester:        newConcurrentHash(digester),
		maximumFileSize: maximumFileSize,
		rootExists:      existsAndIsDirectory(root),
	}
}

// shutdown terminates stager resources.
func (s *stager) shutdown() {
	s.digester.shutdown()
}

// ensurePrefixExists ensures that the specified prefix directory exists within
// the staging root, using a cache to avoid inefficient recreation.
func (s *stager) ensurePrefixExists(prefixByte byte, prefix string) error {
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

// wipe removes the staging root.
func (s *stager) wipe() error {
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

// Sink implements the Sink method of rsync.Sinker.
func (s *stager) Sink(path string) (io.WriteCloser, error) {
	// Create the staging root if we haven't already.
	if !s.rootExists {
		// Attempt to create the root directory.
		if err := os.Mkdir(s.root, 0700); err != nil {
			return nil, fmt.Errorf("unable to create staging root: %w", err)
		}

		// Mark the directory as hidden, if requested.
		if s.hideRoot {
			if err := filesystem.MarkHidden(s.root); err != nil {
				return nil, fmt.Errorf("unable to make staging root as hidden: %w", err)
			}
		}

		// Update our tracking.
		s.rootExists = true
	}

	// Create a temporary storage file in the staging root.
	storage, err := os.CreateTemp(s.root, "staging")
	if err != nil {
		return nil, fmt.Errorf("unable to create temporary storage file: %w", err)
	}

	// Reset the hash function state.
	s.digester.reset()

	// Success.
	return &stagingSink{
		stager:      s,
		path:        path,
		storage:     storage,
		maximumSize: s.maximumFileSize,
	}, nil
}

// Provide implements the Provide method of sync.Provider.
func (s *stager) Provide(path string, digest []byte) (string, error) {
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
