package sync

import (
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"

	"golang.org/x/text/unicode/norm"

	"github.com/golang/protobuf/ptypes"

	fs "github.com/havoc-io/mutagen/pkg/filesystem"
)

const (
	// scannerCopyBufferSize specifies the size of the internal buffer that a
	// scanner uses to copy file data.
	// TODO: Figure out if we should set this on a per-machine basis. This value
	// is taken from Go's io.Copy method, which defaults to allocating a 32k
	// buffer if none is provided.
	scannerCopyBufferSize = 32 * 1024

	// defaultInitialCacheCapacity specifies the default capacity for new
	// filesystem and ignore caches when the corresponding existing cache is nil
	// or empty. It is designed to save several rounds of cache capacity
	// doubling on insert without always allocating a huge cache. Its value is
	// somewhat arbitrary.
	defaultInitialCacheCapacity = 1024
)

// scanner provides the recursive implementation of scanning.
type scanner struct {
	// root is the path to the synchronization root.
	root string
	// hasher is the hashing function to use for computing file digests.
	hasher hash.Hash
	// cache is the existing cache to use for fast digest lookups.
	cache *Cache
	// ignorer is the ignorer identifying ignored paths.
	ignorer *ignorer
	// ignoreCache is the cache of ignored path behavior.
	ignoreCache IgnoreCache
	// symlinkMode is the symlink mode to use for synchronization.
	symlinkMode SymlinkMode
	// newCache is the new file digest cache to populate.
	newCache *Cache
	// newIgnoreCache is the new ignored path behavior cache to populate.
	newIgnoreCache IgnoreCache
	// buffer is the read buffer used for computing file digests.
	buffer []byte
	// deviceID is the device ID of the synchronization root filesystem.
	deviceID uint64
	// recomposeUnicode indicates whether or not filenames need to be recomposed
	// due to Unicode decomposition behavior on the synchronization root
	// filesystem.
	recomposeUnicode bool
	// preservesExecutability indicates whether or not the synchronization root
	// filesystem preserves POSIX executability bits.
	preservesExecutability bool
}

// file performs processing of a file entry. Exactly one of file or parent will
// be non-nil, depending on whether or not the file represents the root path.
// If file is non-nil, this function is responsible for closing it.
func (s *scanner) file(path string, file fs.ReadableFile, metadata *fs.Metadata, parent *fs.Directory) (*Entry, error) {
	// If the file is non-nil, defer its closure.
	if file != nil {
		defer file.Close()
	}

	// Compute executability.
	executable := s.preservesExecutability && anyExecutableBitSet(metadata.Mode)

	// Convert the timestamp to Protocol Buffers format.
	modificationTimeProto, err := ptypes.TimestampProto(metadata.ModificationTime)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert modification time format")
	}

	// Try to find a cached digest. We require that type, modification time,
	// file size, and file ID haven't changed in order to re-use digests. We
	// don't check for permission bit changes since they don't affect content.
	var digest []byte
	cached, hit := s.cache.Entries[path]
	match := hit &&
		(metadata.Mode&fs.ModeTypeMask) == (fs.Mode(cached.Mode)&fs.ModeTypeMask) &&
		modificationTimeProto.Seconds == cached.ModificationTime.Seconds &&
		modificationTimeProto.Nanos == cached.ModificationTime.Nanos &&
		metadata.Size == cached.Size &&
		metadata.FileID == cached.FileID
	if match {
		digest = cached.Digest
	}

	// If we weren't able to pull a digest from the cache, compute one manually.
	if digest == nil {
		// Open the file if it's not open already. If we do open it, then defer
		// its closure.
		if file == nil {
			file, err = parent.OpenFile(metadata.Name)
			if err != nil {
				return nil, errors.Wrap(err, "unable to open file")
			}
			defer file.Close()
		}

		// Reset the hash state.
		s.hasher.Reset()

		// Copy data into the hash and very that we copied as much as expected.
		if copied, err := io.CopyBuffer(s.hasher, file, s.buffer); err != nil {
			return nil, errors.Wrap(err, "unable to hash file contents")
		} else if uint64(copied) != metadata.Size {
			return nil, errors.New("hashed size mismatch")
		}

		// Compute the digest.
		digest = s.hasher.Sum(nil)
	}

	// Add a cache entry.
	s.newCache.Entries[path] = &CacheEntry{
		Mode:             uint32(metadata.Mode),
		ModificationTime: modificationTimeProto,
		Size:             metadata.Size,
		FileID:           metadata.FileID,
		Digest:           digest,
	}

	// Success.
	return &Entry{
		Kind:       EntryKind_File,
		Executable: executable,
		Digest:     digest,
	}, nil
}

// symbolicLink performs processing of a symbolic link entry.
func (s *scanner) symbolicLink(path, name string, parent *fs.Directory, enforcePortable bool) (*Entry, error) {
	// Read the link target.
	target, err := parent.ReadSymbolicLink(name)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read symbolic link target")
	}

	// If requested, enforce that the link is portable, otherwise just ensure
	// that it's non-empty (this is required even in POSIX raw mode).
	if enforcePortable {
		target, err = normalizeSymlinkAndEnsurePortable(path, target)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("invalid symbolic link (%s)", path))
		}
	} else if target == "" {
		return nil, errors.New("symbolic link target is empty")
	}

	// Success.
	return &Entry{
		Kind:   EntryKind_Symlink,
		Target: target,
	}, nil
}

// directory performs processing of a directory entry. Exactly one of directory
// or parent will be non-nil, depending on whether or not the directory
// represents the root path. If directory is non-nil, then this function is
// responsible for closing it.
func (s *scanner) directory(path string, directory *fs.Directory, metadata *fs.Metadata, parent *fs.Directory) (*Entry, error) {
	// If the directory has already been opened, then defer its closure.
	if directory != nil {
		defer directory.Close()
	}

	// Verify that we haven't crossed a directory boundary (which might
	// potentially change executability preservation or Unicode decomposition
	// behavior).
	if metadata.DeviceID != s.deviceID {
		return nil, errors.New("scan crossed filesystem boundary")
	}

	// If the directory is not yet opened, then open it. If we do open it, then
	// defer its closure.
	var err error
	if directory == nil {
		directory, err = parent.OpenDirectory(metadata.Name)
		if err != nil {
			return nil, errors.Wrap(err, "unable to open directory")
		}
		defer directory.Close()
	}

	// Read directory contents.
	directoryContents, err := directory.ReadContents()
	if err != nil {
		return nil, errors.Wrap(err, "unable to read directory contents")
	}

	// RACE: There is technically a race condition here between the listing of
	// directory contents and their processing. This is an inherent reality of
	// our non-atomic synchronization cycles. The worst case fallout is missing
	// file contents (which will be seen during the next synchronization cycle
	// or (if they conflict with changes) later in this synchronization cycle)
	// having stale metadata by which to classify contents (which will result in
	// a scan error), or having contents which have been deleted (which will
	// result in a scan error). This race window is actually slightly
	// advantageous, because it gives us some opportunity to detect concurrent
	// filesystem modifications.

	// Compute entries.
	contents := make(map[string]*Entry, len(directoryContents))
	for _, c := range directoryContents {
		// Extract the content name.
		name := c.Name

		// If this is an intermediate temporary file, then ignore it.
		if fs.IsTemporaryFileName(name) {
			continue
		}

		// Recompose Unicode in the content name if necessary.
		if s.recomposeUnicode {
			name = norm.NFC.String(name)
		}

		// Compute the content path.
		contentPath := pathJoin(path, name)

		// Compute the kind for this content, skipping if unsupported.
		var kind EntryKind
		switch c.Mode & fs.ModeTypeMask {
		case fs.ModeTypeDirectory:
			kind = EntryKind_Directory
		case fs.ModeTypeFile:
			kind = EntryKind_File
		case fs.ModeTypeSymbolicLink:
			kind = EntryKind_Symlink
		default:
			continue
		}

		// Determine whether or not this path is ignored and update the new
		// ignore cache.
		isDirectory := kind == EntryKind_Directory
		ignoreCacheKey := IgnoreCacheKey{contentPath, isDirectory}
		ignored, ok := s.ignoreCache[ignoreCacheKey]
		if !ok {
			ignored = s.ignorer.ignored(contentPath, isDirectory)
		}
		s.newIgnoreCache[ignoreCacheKey] = ignored
		if ignored {
			continue
		}

		// Handle based on kind.
		var entry *Entry
		if kind == EntryKind_File {
			entry, err = s.file(contentPath, nil, c, directory)
		} else if kind == EntryKind_Symlink {
			if s.symlinkMode == SymlinkMode_SymlinkPortable {
				entry, err = s.symbolicLink(contentPath, name, directory, true)
			} else if s.symlinkMode == SymlinkMode_SymlinkIgnore {
				continue
			} else if s.symlinkMode == SymlinkMode_SymlinkPOSIXRaw {
				entry, err = s.symbolicLink(contentPath, name, directory, false)
			} else {
				panic("unsupported symlink mode")
			}
		} else if kind == EntryKind_Directory {
			entry, err = s.directory(contentPath, nil, c, directory)
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

// Scan provides recursive filesystem scanning facilities for synchronization
// roots.
func Scan(root string, hasher hash.Hash, cache *Cache, ignores []string, ignoreCache IgnoreCache, symlinkMode SymlinkMode) (*Entry, bool, bool, *Cache, IgnoreCache, error) {
	// A nil cache is technically valid, but if the provided cache is nil,
	// replace it with an empty one, that way we don't have to use the
	// GetEntries accessor everywhere.
	if cache == nil {
		cache = &Cache{}
	}

	// Create the ignorer.
	ignorer, err := newIgnorer(ignores)
	if err != nil {
		return nil, false, false, nil, nil, errors.Wrap(err, "unable to create ignorer")
	}

	// Verify that the symlink mode is valid for this platform.
	if symlinkMode == SymlinkMode_SymlinkPOSIXRaw && runtime.GOOS == "windows" {
		return nil, false, false, nil, nil, errors.New("raw POSIX symlinks not supported on Windows")
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

	// Create a new ignore cache to populate. Estimate its capacity based on the
	// existing ignore cache length. If the existing cache is empty, create one
	// with the default capacity.
	initialIgnoreCacheCapacity := defaultInitialCacheCapacity
	if ignoreCacheLength := len(ignoreCache); ignoreCacheLength != 0 {
		initialIgnoreCacheCapacity = ignoreCacheLength
	}
	newIgnoreCache := make(IgnoreCache, initialIgnoreCacheCapacity)

	// Create a scanner.
	s := &scanner{
		root:           root,
		hasher:         hasher,
		cache:          cache,
		ignorer:        ignorer,
		ignoreCache:    ignoreCache,
		symlinkMode:    symlinkMode,
		newCache:       newCache,
		newIgnoreCache: newIgnoreCache,
		buffer:         make([]byte, scannerCopyBufferSize),
	}

	// Open the root. We explicitly disallow symbolic links at the root path,
	// though intermediate symbolic links are fine.
	rootObject, metadata, err := fs.Open(root, false)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, false, newCache, newIgnoreCache, nil
		} else {
			return nil, false, false, nil, nil, errors.Wrap(err, "unable to probe scan root")
		}
	}

	// Store the device ID for the root.
	s.deviceID = metadata.DeviceID

	// Handle the scan based on the root type.
	if rootType := metadata.Mode & fs.ModeTypeMask; rootType == fs.ModeTypeDirectory {
		// Extract the directory object.
		rootDirectory, ok := rootObject.(*fs.Directory)
		if !ok {
			panic("invalid directory object returned from root open operation")
		}

		// Probe and set Unicode decomposition behavior for the root directory.
		if decomposes, err := fs.DecomposesUnicode(rootDirectory); err != nil {
			rootDirectory.Close()
			return nil, false, false, nil, nil, errors.Wrap(err, "unable to probe root Unicode decomposition behavior")
		} else {
			s.recomposeUnicode = decomposes
		}

		// Probe and set executability preservation behavior for the root
		// directory.
		if preserves, err := fs.PreservesExecutability(rootDirectory); err != nil {
			rootDirectory.Close()
			return nil, false, false, nil, nil, errors.Wrap(err, "unable to probe root executability preservation behavior")
		} else {
			s.preservesExecutability = preserves
		}

		// Perform a recursive scan. The directory function is responsible for
		// closing the directory object, regardless of errors.
		if rootEntry, err := s.directory("", rootDirectory, metadata, nil); err != nil {
			return nil, false, false, nil, nil, err
		} else {
			return rootEntry, s.preservesExecutability, s.recomposeUnicode, newCache, newIgnoreCache, nil
		}
	} else if rootType == fs.ModeTypeFile {
		// Extract the file object.
		rootFile, ok := rootObject.(fs.ReadableFile)
		if !ok {
			panic("invalid file object returned from root open operation")
		}

		// Probe and set executability preservation behavior for the parent of
		// the root path.
		//
		// RACE: There is technically a race condition here on POSIX because the
		// root file that we have open may have been unlinked and the parent
		// directory path removed or replaced. Even if the file hasn't been
		// unlinked, we still have to make this probe by path since there's no
		// way (both due to API and underlying design) to grab a parent
		// directory by file descriptor on POSIX (it's not a well-defined
		// concept (due at least to the existence of hard links)). Anyway, this
		// race does not have any significant risk. The only other option would
		// be to switch executability preservation detection to use fstatfs, but
		// this is a non-standardized call that returns different metadata on
		// each platform. Even on platforms where detailed metadata is provided,
		// the filesystem identifier alone may not be enough to determine this
		// behavior.
		if preserves, err := fs.PreservesExecutabilityByPath(filepath.Dir(root)); err != nil {
			rootFile.Close()
			return nil, false, false, nil, nil, errors.Wrap(err, "unable to probe root parent executability preservation behavior")
		} else {
			s.preservesExecutability = preserves
		}

		// Perform a scan of the root file. The file function is responsible for
		// closing the file object, regardless of errors.
		if rootEntry, err := s.file("", rootFile, metadata, nil); err != nil {
			return nil, false, false, nil, nil, err
		} else {
			return rootEntry, s.preservesExecutability, false, newCache, newIgnoreCache, nil
		}
	} else {
		panic("invalid type returned from root open operation")
	}
}
