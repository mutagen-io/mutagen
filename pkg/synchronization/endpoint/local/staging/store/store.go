package store

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/zeebo/xxh3"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/must"
	"github.com/mutagen-io/mutagen/pkg/stream"
)

var (
	// errStoreUninitialized is returned when methods are invoked on a store
	// before it is initialized with Initialize.
	errStoreUninitialized = errors.New("store uninitialized")
	// errDigestEmpty is returned when an empty digest is provided or a hash
	// function generates an empty digest.
	errDigestEmpty = errors.New("digest empty")
)

const (
	// storageWriteBufferSize is the buffer size to use for storage writes.
	storageWriteBufferSize = 64 * 1024
)

// Store implements content-addressable storage for staging files. In addition
// to standard CAS addressing, it adds an additional level of addressing based
// on the expected path for content within the synchronization root. After
// Initialize is called, the Allocate, Contains, and Path methods may be invoked
// concurrently, and the Storage instances returned by Allocate may be used
// concurrently. Initialize and Finalize may never be called concurrently with
// any other methods or while outstanding Storage instances (not finalized with
// Commit or Discard) exist.
type Store struct {
	// root is the path to the directory used for storage.
	root string
	// hidden indicates whether or not the root storage directory should be
	// hidden on the filesystem.
	hidden bool
	// maximumFileSize is the maximum allowed size for a single storage file.
	maximumFileSize uint64
	// writeBufferPool is a pool of bufio.Writer for buffering storage writes.
	// When not in use, their writer is set to io.Discard.
	writeBufferPool sync.Pool
	// contentHasherPool is a pool of hash.Hash for computing content digests.
	contentHasherPool sync.Pool
	// pathHasherPool is a pool of hash.Hash for computing path digests.
	pathHasherPool sync.Pool
	// initialized indicates whether or not the store has been initialized. If
	// this field is true, then the root directory exists and the values in
	// prefixExists are correct. If this field is false, then the state of the
	// root directory is unknown and the values in prefixExists must be
	// considered invalid.
	initialized bool
	// prefixLock serializes creation of prefix directories and modifications to
	// prefixExists. Holding the lock is only required for concurrent-safe
	// sections of Store-related code.
	prefixLock sync.RWMutex
	// prefixExists tracks whether or not individual prefix directories exist.
	// It is indexed on the byte value corresponding to the prefix directory.
	prefixExists [256]bool

	logger *logging.Logger
}

// NewStore creates a new store instance with the specified parameters.
func NewStore(root string, hidden bool, maximumFileSize uint64, contentHasherFactory func() hash.Hash, logger *logging.Logger) *Store {
	return &Store{
		root:            root,
		hidden:          hidden,
		maximumFileSize: maximumFileSize,
		writeBufferPool: sync.Pool{
			New: func() any {
				return bufio.NewWriterSize(io.Discard, storageWriteBufferSize)
			},
		},
		contentHasherPool: sync.Pool{
			New: func() any {
				return contentHasherFactory()
			},
		},
		pathHasherPool: sync.Pool{
			New: func() any {
				return xxh3.New()
			},
		},
		logger: logger,
	}
}

// isLowerCaseHexCharacter indicates whether or not a byte represents a
// character that might appear in a lower-case hex encoding.
func isLowerCaseHexCharacter(c byte) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f')
}

// parsePrefixDirectoryName parses a prefix directory name, returning its byte
// value and a boolean indicating whether or not the name was valid for a prefix
// directory.
func parsePrefixDirectoryName(name string) (byte, bool) {
	// Verify that the name is a valid prefix directory name.
	valid := len(name) == 2 &&
		isLowerCaseHexCharacter(name[0]) && isLowerCaseHexCharacter(name[1])
	if !valid {
		return 0, false
	}

	// Parse the name.
	var result [1]byte
	if n, err := hex.Decode(result[:], []byte(name)); n != 1 || err != nil {
		panic("decode failed on valid hexadecimal name")
	}

	// Success.
	return result[0], true
}

// Initialize prepares the store to receive content. It must be called before
// any calls to Allocate, Contains, or Path, though Finalize can be called
// without first calling Initialize.
func (s *Store) Initialize() error {
	// If the store is already initialized, then there's nothing we need to do.
	if s.initialized {
		return nil
	}

	// Attempt to create the storage root. If we create it, then hide it if
	// necessary. If it already exists, then ensure that it's a directory,
	// otherwise we'll have to abort.
	var existed bool
	if err := os.Mkdir(s.root, 0700); err != nil {
		if errors.Is(err, fs.ErrExist) {
			if metadata, err := os.Lstat(s.root); err != nil {
				return fmt.Errorf("unable to query existing storage root: %w", err)
			} else if !metadata.IsDir() {
				return errors.New("storage root exists and is not directory")
			}
			existed = true
		} else {
			return fmt.Errorf("unable to create storage root: %w", err)
		}
	} else if s.hidden {
		if err := filesystem.MarkHidden(s.root); err != nil {
			return fmt.Errorf("unable to hide storage root: %w", err)
		}
	}

	// Reset the prefix existence tracker.
	s.prefixExists = [256]bool{}

	// If the prefix already existed, then scan its contents to look for
	// existing prefix directories. If this fails, then we just return an error,
	// but we don't remove the directory since it will have existed already and
	// we won't have made any changes to it.
	if existed {
		contents, err := os.ReadDir(s.root)
		if err != nil {
			return fmt.Errorf("unable to read existing storage root contents: %w", err)
		}
		for _, content := range contents {
			if p, ok := parsePrefixDirectoryName(content.Name()); !ok {
				continue
			} else if !content.IsDir() {
				return fmt.Errorf("non-directory content with prefix name (%s) found in storage root", content.Name())
			} else {
				s.prefixExists[p] = true
			}
		}
	}

	// Mark the store as initialized.
	s.initialized = true

	// Success.
	return nil
}

// Allocate allocates temporary storage for receiving data.
func (s *Store) Allocate(logger *logging.Logger) (*Storage, error) {
	// Verify that the store is initialized.
	if !s.initialized {
		return nil, errStoreUninitialized
	}

	// Create a temporary storage file in the staging root.
	storage, err := os.CreateTemp(s.root, "storage")
	if err != nil {
		return nil, fmt.Errorf("unable to create temporary storage file: %w", err)
	}

	// Acquire and reset a hasher that we can use to digest content.
	hasher := s.contentHasherPool.Get().(hash.Hash)
	hasher.Reset()

	// Create a hashed writer targeting storage.
	writer := stream.NewHashedWriter(storage, hasher)

	// Acquire and reset a write buffer to target the writer.
	buffer := s.writeBufferPool.Get().(*bufio.Writer)
	buffer.Reset(writer)

	// Success.
	return &Storage{
		store:   s,
		storage: storage,
		hasher:  hasher,
		writer:  writer,
		buffer:  buffer,
		logger:  logger,
	}, nil
}

// target computes the storage destination path for content with the specified
// path and digest. Callers must verify that the digest is non-empty, otherwise
// this method will panic. It returns the target path and associated prefix
// directory name. It does not attempt to create the prefix directory. This
// method is safe for concurrent invocation.
func (s *Store) target(path string, digest []byte) (string, string) {
	// Convert the digest to hexadecimal encoding and extract the prefix.
	digestHex := hex.EncodeToString(digest)
	prefix := digestHex[:2]

	// Grab a path hasher and ensure that it's reset.
	pathHasher := s.pathHasherPool.Get().(*xxh3.Hasher)
	pathHasher.Reset()

	// Compute the path digest.
	must.WriteString(pathHasher, path, s.logger)
	pathDigest := pathHasher.Sum128()

	// Return the path hasher to the pool.
	s.pathHasherPool.Put(pathHasher)

	// Compute the hexadecimal encoded digest of the path name.
	pathDigestBytes := pathDigest.Bytes()
	pathDigestHex := hex.EncodeToString(pathDigestBytes[:])

	// Compute the storage name.
	storageName := digestHex + pathDigestHex

	// Success.
	return filepath.Join(s.root, prefix, storageName), prefix
}

// Contains returns whether or not the store contains the specified content.
func (s *Store) Contains(path string, digest []byte) (bool, error) {
	// Verify that the store is initialized.
	if !s.initialized {
		return false, errStoreUninitialized
	}

	// Verify that the digest is non-empty.
	if len(digest) == 0 {
		return false, errDigestEmpty
	}

	// Check if the corresponding prefix directory exists. If not, then we know
	// that the content couldn't possibly exist.
	s.prefixLock.RLock()
	prefixExists := s.prefixExists[digest[0]]
	s.prefixLock.RUnlock()
	if !prefixExists {
		return false, nil
	}

	// Compute the storage path for the content.
	target, _ := s.target(path, digest)

	// Check if the context exists. If it does, but isn't a regular file, then
	// just pretend that it doesn't exist, because any storage that targets that
	// location will simply replace it.
	if metadata, err := os.Lstat(target); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("unable to query storage: %w", err)
	} else if metadata.Mode()&fs.ModeType != 0 {
		return false, nil
	}

	// Success.
	return true, nil
}

// Path provides the storage path for the specified addressing parameters. It
// does not verify that the content exists. Callers should instead verify
// existence via Contains or by storing the specified content.
func (s *Store) Path(path string, digest []byte) (string, error) {
	// Verify that the store is initialized.
	if !s.initialized {
		return "", errStoreUninitialized
	}

	// Verify that the digest is non-empty.
	if len(digest) == 0 {
		return "", errDigestEmpty
	}

	// Compute the storage path for the content.
	target, _ := s.target(path, digest)

	// Success.
	return target, nil
}

// Finalize remove's the store's on-disk content and resets its internal state.
// After calling Finalize, the Initialize method must be called before the Store
// can be used again.
func (s *Store) Finalize() error {
	// Mark the store as no longer initialized. We do this first just in case
	// the store removal (which isn't atomic) only partially completes.
	s.initialized = false

	// Remove the store root.
	if err := os.RemoveAll(s.root); err != nil {
		return fmt.Errorf("unable to remove storage root: %w", err)
	}

	// Success.
	return nil
}

// Storage represents a temporarily allocated receptical for data that should be
// committed to a store.
type Storage struct {
	// store is the parent store for the storage.
	store *Store
	// storage is the temporary file being used to store data.
	storage *os.File
	// hasher computes the digest of the storage content.
	hasher hash.Hash
	// writer is the hashed writer targeting storage and hasher.
	writer io.Writer
	// buffer is the write buffer targeting writer.
	buffer *bufio.Writer
	// currentSize is the number of bytes that have been written to the file.
	currentSize uint64

	logger *logging.Logger
}

// Write implements io.Writer.Write for the storage.
func (s *Storage) Write(data []byte) (int, error) {
	// Watch for size violations.
	if (s.store.maximumFileSize - s.currentSize) < uint64(len(data)) {
		return 0, errors.New("maximum file size reached")
	}

	// Write to the buffer.
	n, err := s.buffer.Write(data)

	// Update the current size. We needn't worry about this overflowing, because
	// the check above is sufficient to ensure that this amount of data won't
	// overflow the maximum uint64 value.
	s.currentSize += uint64(n)

	// Done.
	return n, err
}

// Commit closes the storage and commits the data to the store, with an address
// computed by a combination of the content digest and the specified path.
func (s *Storage) Commit(path string) error {
	// Close the underlying storage.
	if err := s.buffer.Flush(); err != nil {
		return fmt.Errorf("unable to flush content to disk: %w", err)
	} else if err = s.storage.Close(); err != nil {
		return fmt.Errorf("unable to close underlying storage: %w", err)
	}

	// Compute the final content digest.
	digest := s.hasher.Sum(nil)

	// Return the buffer to the pool.
	s.buffer.Reset(io.Discard)
	s.store.writeBufferPool.Put(s.buffer)

	// Return the hasher to the pool.
	s.store.contentHasherPool.Put(s.hasher)

	// Verify that the content digest has sufficient length.
	if len(digest) == 0 {
		must.OSRemove(s.storage.Name(), s.logger)
		return errDigestEmpty
	}

	// Compute the prefix byte and storage path for the content.
	prefixByte := digest[0]
	target, prefix := s.store.target(path, digest)

	// Ensure that the prefix directory exists.
	s.store.prefixLock.Lock()
	if !s.store.prefixExists[prefixByte] {
		if err := os.Mkdir(filepath.Join(s.store.root, prefix), 0700); err != nil {
			s.store.prefixLock.Unlock()
			must.OSRemove(s.storage.Name(), s.logger)
			return fmt.Errorf("unable to create prefix directory (%s): %w", prefix, err)
		}
		s.store.prefixExists[prefixByte] = true
	}
	s.store.prefixLock.Unlock()

	// Relocate the temporary file to its target destination.
	if err := filesystem.Rename(nil, s.storage.Name(), nil, target, true); err != nil {
		must.OSRemove(s.storage.Name(), s.logger)
		return fmt.Errorf("unable to relocate storage: %w", err)
	}

	// Success.
	return nil
}

// Discard closes the storage and discards the recorded data.
func (s *Storage) Discard() error {
	// Close the underlying storage.
	if err := s.storage.Close(); err != nil {
		return fmt.Errorf("unable to close underlying storage: %w", err)
	}

	// Return the buffer to the pool.
	s.buffer.Reset(io.Discard)
	s.store.writeBufferPool.Put(s.buffer)

	// Return the hasher to the pool.
	s.store.contentHasherPool.Put(s.hasher)

	// Remove the file.
	return os.Remove(s.storage.Name())
}
