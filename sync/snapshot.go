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
	// TODO: Figure out how we want to handle permissions and executability.
	size := uint64(info.Size())
	modificationTime := info.ModTime()
	mode := info.Mode()
	executable := (mode&0111 != 0)

	// Perform a cache lookup and see if we can find a cached digest. If we find
	// a match, propagate the cache entry and re-use the digest.
	cached, hit := s.cache.Entries[target]
	match := hit &&
		cached.Size_ == size &&
		cached.ModificationTime != nil &&
		cached.ModificationTime.Equal(modificationTime) &&
		(os.FileMode(cached.Mode)&os.ModeType) == (mode&os.ModeType)
	if match {
		s.newCache.Entries[target] = cached
		return &Entry{EntryKind_File, executable, cached.Digest, nil}, nil
	}

	// If we couldn't find a cached digest, compute a full one.
	file, err := os.Open(filepath.Join(s.root, target))
	if err != nil {
		return nil, errors.Wrap(err, "unable to open file")
	}
	defer file.Close()
	s.hasher.Reset()
	if copied, err := io.CopyBuffer(s.hasher, file, s.buffer); err != nil {
		return nil, errors.Wrap(err, "unable to hash file contents")
	} else if uint64(copied) != size {
		return nil, errors.New("hashed size mismatch")
	}
	digest := s.hasher.Sum(nil)

	// Add a cache entry.
	s.newCache.Entries[target] = &CacheEntry{
		size,
		uint32(mode),
		&modificationTime,
		digest,
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
			return nil, nil, nil
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
