package sync

import (
	"fmt"
	"hash"
	"io"
	"os"
	pathpkg "path"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"

	"github.com/golang/protobuf/ptypes"

	"github.com/havoc-io/mutagen/pkg/filesystem"
)

const (
	// scannerCopyBufferSize specifies the size of the internal buffer that a
	// scanner uses to copy file data.
	// TODO: Figure out if we should set this on a per-machine basis. This value
	// is taken from Go's io.Copy method, which defaults to allocating a 32k
	// buffer if none is provided.
	scannerCopyBufferSize = 32 * 1024

	// defaultInitialCacheCapacity specifies the default capacity for new caches
	// when the existing cache is nil or empty. It is designed to save several
	// rounds of cache capacity doubling on insert without always allocating a
	// huge cache. Its value is somewhat arbitrary.
	defaultInitialCacheCapacity = 1024
)

type scanner struct {
	root     string
	hasher   hash.Hash
	cache    *Cache
	ignorer  *ignorer
	newCache *Cache
	buffer   []byte
}

func (s *scanner) file(path string, info os.FileInfo) (*Entry, error) {
	// Extract metadata.
	mode := info.Mode()
	modificationTime := info.ModTime()
	size := uint64(info.Size())

	// Compute executability.
	executable := (mode&AnyExecutablePermission != 0)

	// Convert the timestamp to Protocol Buffers format.
	modificationTimeProto, err := ptypes.TimestampProto(modificationTime)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert modification time format")
	}

	// Try to find a cached digest. We only enforce that type, modification
	// time, and size haven't changed in order to re-use digests.
	var digest []byte
	cached, hit := s.cache.Entries[path]
	match := hit &&
		(os.FileMode(cached.Mode)&os.ModeType) == (mode&os.ModeType) &&
		cached.ModificationTime != nil &&
		modificationTimeProto.Seconds == cached.ModificationTime.Seconds &&
		modificationTimeProto.Nanos == cached.ModificationTime.Nanos &&
		cached.Size == size
	if match {
		digest = cached.Digest
	}

	// If we weren't able to pull a digest from the cache, compute one manually.
	if digest == nil {
		// Open the file and ensure its closure.
		file, err := os.Open(filepath.Join(s.root, path))
		if err != nil {
			return nil, errors.Wrap(err, "unable to open file")
		}
		defer file.Close()

		// Reset the hash state.
		s.hasher.Reset()

		// Copy data into the hash and very that we copied as much as expected.
		if copied, err := io.CopyBuffer(s.hasher, file, s.buffer); err != nil {
			return nil, errors.Wrap(err, "unable to hash file contents")
		} else if uint64(copied) != size {
			return nil, errors.New("hashed size mismatch")
		}

		// Compute the digest.
		digest = s.hasher.Sum(nil)
	}

	// Add a cache entry.
	s.newCache.Entries[path] = &CacheEntry{
		Mode:             uint32(mode),
		ModificationTime: modificationTimeProto,
		Size:             size,
		Digest:           digest,
	}

	// Success.
	return &Entry{
		Kind:       EntryKind_File,
		Executable: executable,
		Digest:     digest,
	}, nil
}

func (s *scanner) symlink(path string, enforcePortable bool) (*Entry, error) {
	// Read the link target.
	target, err := os.Readlink(filepath.Join(s.root, path))
	if err != nil {
		return nil, errors.Wrap(err, "unable to read symlink target")
	}

	// If requested, enforce that the link is portable, otherwise just ensure
	// that it's non-empty (this is required even in POSIX raw mode).
	if enforcePortable {
		target, err = normalizeSymlinkAndEnsurePortable(path, target)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("invalid symlink (%s)", path))
		}
	} else if target == "" {
		return nil, errors.New("symlink target is empty")
	}

	// Success.
	return &Entry{
		Kind:   EntryKind_Symlink,
		Target: target,
	}, nil
}

func (s *scanner) directory(path string, symlinkMode SymlinkMode) (*Entry, error) {
	// Read directory contents.
	directoryContents, err := filesystem.DirectoryContents(filepath.Join(s.root, path))
	if err != nil {
		return nil, errors.Wrap(err, "unable to read directory contents")
	}

	// Compute entries.
	contents := make(map[string]*Entry, len(directoryContents))
	for _, name := range directoryContents {
		// Compute the content path.
		contentPath := pathpkg.Join(path, name)

		// Grab stat information for this path. If the path has disappeared
		// between list time and stat time, then concurrent modifications
		// (deletions or renames) are likely occurring and we should abort. We
		// could do what something like filepath.Walk does and check the error
		// using os.IsNotExist and just ignore the file if it's been removed,
		// but in order for our root deletion safety check to be more effective,
		// we don't want to take snapshots in the middle of a deletion
		// operation. It wouldn't stop the snapshot from being correct in the
		// context of our synchronization algorithm, but we really want to get a
		// picture once the deletion has stopped. Again, this doesn't guarantee
		// we'll catch all concurrent deletions - there's a race there, but it
		// will astronomically increase our chances, and also probably minimize
		// the number of resynchronizations that we need to do.
		info, err := os.Lstat(filepath.Join(s.root, contentPath))
		if err != nil {
			return nil, errors.Wrap(err, "unable to stat directory content")
		}

		// Compute the kind for this content, skipping if unsupported.
		kind := EntryKind_File
		if mode := info.Mode(); mode&os.ModeDir != 0 {
			kind = EntryKind_Directory
		} else if mode&os.ModeSymlink != 0 {
			kind = EntryKind_Symlink
		} else if mode&os.ModeType != 0 {
			continue
		}

		// If this path is ignored, then skip it.
		if s.ignorer.ignored(contentPath, kind == EntryKind_Directory) {
			continue
		}

		// Handle based on kind.
		var entry *Entry
		if kind == EntryKind_File {
			entry, err = s.file(contentPath, info)
		} else if kind == EntryKind_Symlink {
			if symlinkMode == SymlinkMode_SymlinkPortable {
				entry, err = s.symlink(contentPath, true)
			} else if symlinkMode == SymlinkMode_SymlinkIgnore {
				continue
			} else if symlinkMode == SymlinkMode_SymlinkPOSIXRaw {
				entry, err = s.symlink(contentPath, false)
			} else {
				panic("unsupported symlink mode")
			}
		} else if kind == EntryKind_Directory {
			entry, err = s.directory(contentPath, symlinkMode)
		} else {
			panic("unhandled entry kind")
		}

		// Watch for errors and add the entry.
		if err != nil {
			return nil, err
		}

		// Add the content.
		contents[name] = entry
	}

	// Success.
	return &Entry{
		Kind:     EntryKind_Directory,
		Contents: contents,
	}, nil
}

// TODO: Note that the provided cache is assumed to be valid (i.e. that it
// doesn't have any nil entries), so callers should run EnsureValid on anything
// they pull from disk
func Scan(root string, hasher hash.Hash, cache *Cache, ignores []string, symlinkMode SymlinkMode) (*Entry, *Cache, error) {
	// A nil cache is technically valid, but if the provided cache is nil,
	// replace it with an empty one, that way we don't have to use the
	// GetEntries accessor everywhere.
	if cache == nil {
		cache = &Cache{}
	}

	// Create the ignorer.
	ignorer, err := newIgnorer(ignores)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to create ignorer")
	}

	// Verify that the symlink mode is valid for this platform.
	if symlinkMode == SymlinkMode_SymlinkPOSIXRaw && runtime.GOOS == "windows" {
		return nil, nil, errors.New("raw POSIX symlinks not supported on Windows")
	}

	// Create a new cache to populate. Estimate its capacity based on the
	// existing cache length. If the existing cache is empty, create one with
	// the default capacity.
	initialCacheCapacity := defaultInitialCacheCapacity
	if cacheLength := len(cache.Entries); cacheLength != 0 {
		initialCacheCapacity = cacheLength
	}
	newCache := &Cache{
		Entries: make(map[string]*CacheEntry, initialCacheCapacity),
	}

	// Create a scanner.
	s := &scanner{
		root:     root,
		hasher:   hasher,
		cache:    cache,
		ignorer:  ignorer,
		newCache: newCache,
		buffer:   make([]byte, scannerCopyBufferSize),
	}

	// Create the snapshot.
	if info, err := os.Lstat(root); err != nil {
		if os.IsNotExist(err) {
			return nil, newCache, nil
		} else {
			return nil, nil, errors.Wrap(err, "unable to probe scan root")
		}
	} else if mode := info.Mode(); mode&os.ModeDir != 0 {
		if rootEntry, err := s.directory("", symlinkMode); err != nil {
			return nil, nil, err
		} else {
			return rootEntry, newCache, nil
		}
	} else if mode&os.ModeType != 0 {
		// We explicitly disallow symlinks as synchronization roots because
		// there's no easy way to propagate changes to them.
		return nil, nil, errors.New("invalid scan root type")
	} else {
		if rootEntry, err := s.file("", info); err != nil {
			return nil, nil, err
		} else {
			return rootEntry, newCache, nil
		}
	}
}
