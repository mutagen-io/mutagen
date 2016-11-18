package sync

import (
	"hash"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// TODO: Figure out if we should set this on a per-machine basis. This value is
// taken from Go's io.Copy method, which defaults to allocating a 32k buffer if
// none is provided.
const snapshotterCopyBufferSize = 32 * 1024

type snapshotter struct {
	root     string
	hasher   hash.Hash
	cache    *Cache
	newCache *Cache
	buffer   []byte
	// TODO: Add ignore stack. Probably best to use something like
	// https://github.com/sabhiram/go-git-ignore for the implementation. Also
	// have a look at the forks for that repository, some are a bit cleaner and
	// one supports base paths, though I don't know if we need or want that.
}

func (s *snapshotter) file(target string, info os.FileInfo) (*Entry, error) {
	// Extract metadata.
	mode := info.Mode()
	modificationTime := info.ModTime()
	size := uint64(info.Size())

	// Compute executability.
	executable := (mode&0111 != 0)

	// Try to find a cached digest. We only enforce that type, modification
	// time, and size haven't changed in order to re-use digests.
	var digest []byte
	cached, hit := s.cache.Entries[target]
	// TODO: We should add another condition to match that enforces modification
	// time is before the timestamp of the cache on disk. This is the same
	// solution that Git uses to solve its index race condition. Update the
	// comment above once this is done.
	match := hit &&
		(os.FileMode(cached.Mode)&os.ModeType) == (mode&os.ModeType) &&
		cached.ModificationTime != nil &&
		cached.ModificationTime.Equal(modificationTime) &&
		cached.Size_ == size
	if match {
		digest = cached.Digest
	}

	// If we weren't able to pull a digest from the cache, compute one manually.
	if digest == nil {
		// Open the file and ensure its closure.
		file, err := os.Open(filepath.Join(s.root, target))
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
	s.newCache.Entries[target] = &CacheEntry{
		Mode:             uint32(mode),
		ModificationTime: &modificationTime,
		Size_:            size,
		Digest:           digest,
	}

	// Success.
	return &Entry{EntryKind_File, executable, digest, nil}, nil
}

func (s *snapshotter) directory(target string) (*Entry, error) {
	// Read directory contents.
	directoryContents, err := ioutil.ReadDir(filepath.Join(s.root, target))
	if err != nil {
		return nil, errors.Wrap(err, "unable to read directory")
	}

	// TODO: Look for a .mutagenignore and push onto ignore stack.

	// Compute entries.
	contents := make(map[string]*Entry, len(directoryContents))
	for _, info := range directoryContents {
		// Compute the kind for this content, skipping if unsupported.
		kind := EntryKind_File
		if mode := info.Mode(); mode&os.ModeDir != 0 {
			kind = EntryKind_Directory
		} else if mode&os.ModeType != 0 {
			continue
		}

		// Compute the base name and relative path for this content.
		contentName := info.Name()
		contentTarget := filepath.Join(target, contentName)

		// TODO: Check if this entry is ignored and skip if so.

		// Handle based on kind.
		var entry *Entry
		if kind == EntryKind_File {
			entry, err = s.file(contentTarget, info)
		} else if kind == EntryKind_Directory {
			entry, err = s.directory(contentTarget)
		} else {
			panic("unhandled entry kind")
		}

		// Watch for errors and add the entry.
		if err != nil {
			return nil, err
		}

		// Add the content.
		contents[contentName] = entry
	}

	// TODO: Pop ignore stack.

	// Success.
	return &Entry{EntryKind_Directory, false, nil, contents}, nil
}

func Snapshot(root string, hasher hash.Hash, cache *Cache) (*Entry, *Cache, error) {
	// If the cache is nil, create an empty one.
	if cache == nil {
		cache = &Cache{}
	}

	// Create a new cache to populate.
	newCache := &Cache{make(map[string]*CacheEntry)}

	// Create a snapshotter.
	s := &snapshotter{
		root:     root,
		hasher:   hasher,
		cache:    cache,
		newCache: newCache,
		buffer:   make([]byte, snapshotterCopyBufferSize),
	}

	// Create the snapshot.
	if info, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return nil, newCache, nil
		} else {
			return nil, nil, errors.Wrap(err, "unable to probe snapshot root")
		}
	} else if mode := info.Mode(); mode&os.ModeDir != 0 {
		if snapshot, err := s.directory(""); err != nil {
			return nil, nil, err
		} else {
			return snapshot, newCache, nil
		}
	} else if mode&os.ModeType != 0 {
		return nil, nil, errors.New("invalid snapshot root type")
	} else {
		if snapshot, err := s.file("", info); err != nil {
			return nil, nil, err
		} else {
			return snapshot, newCache, nil
		}
	}
}
