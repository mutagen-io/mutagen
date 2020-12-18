package core

import (
	"context"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"golang.org/x/text/unicode/norm"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/filesystem/behavior"
)

const (
	// scannerCopyBufferSize specifies the size of the internal buffer that a
	// scanner uses to copy file data.
	// TODO: Figure out if we should set this on a per-machine basis. This value
	// is taken from Go's io.Copy method, which defaults to allocating a 32k
	// buffer if none is provided.
	scannerCopyBufferSize = 32 * 1024

	// scannerCopyPreemptionInterval specifies the interval between preemption
	// checks when performing digest writes. This, multiplied by
	// scannerCopyBufferSize, determines the maximum number of bytes that can be
	// written to a digest between preemption checks and thus controls the
	// maximum preemption latency.
	scannerCopyPreemptionInterval = 1024

	// defaultInitialCacheCapacity specifies the default capacity for new
	// filesystem and ignore caches when the corresponding existing cache is nil
	// or empty. It is designed to save several rounds of cache capacity
	// doubling on insert without always allocating a huge cache. Its value is
	// somewhat arbitrary.
	defaultInitialCacheCapacity = 1024
)

var (
	// errScanCancelled indicates that the scan was cancelled.
	errScanCancelled = errors.New("scan cancelled")
)

// behaviorCache is a cache mapping filesystem device IDs to behavioral
// information. It is only used in cases where probe files are required for
// probing behavior, because those cases are (a) more expensive and (b) cause
// watching/scanning feedback loops with synchronization endpoints if not
// cached. For cases where filesystem behavior is assumed or probed via fstatfs,
// there's no need to cache the information since (a) it's relatively cheap and
// (b) it won't cause watching/scanning feedback loops since it doesn't perturb
// the filesystem.
//
// HACK: This cache is really a hack and something of a layering violation. Its
// purpose isn't really optimization by avoidance of probe files (which is a
// nice side-effect), but rather avoidance of synchronization endpoint
// watching/scanning feedback loops caused by probe files. The fact that it
// really only exists for this latter reason indicates some knowledge of how
// synchronization endpoints behave. The implementation of this cache is also
// a layering violation in the sense that we rely on it not being used on
// Windows because we don't actually compute real device IDs on Windows and
// would thus have cache collisions. Fortunately, we know that it won't be used
// on Windows because probe files aren't used on Windows. A more "correct"
// approach would probably be to have the scan return behavioral information and
// information about whether or not probe files were used to the endpoint, and
// then accept cached behavioral information from the endpoint (which *should*
// be allowed to know about both probe files and filesystem watching). But the
// Scan function and endpoint implementation are already so complex that this
// makes code significantly more cumbersome and fragile, so in the end this
// layering violation is the lesser evil. Eventually we'll get rid of probe
// files and the need for this cache will go away.
var behaviorCache struct {
	sync.RWMutex
	// preservesExecutability maps device IDs to executability preservation
	// behavior.
	preservesExecutability map[uint64]bool
	// decomposesUnicode maps device IDs to Unicode decomposition behavior.
	decomposesUnicode map[uint64]bool
}

func init() {
	// Initialize the behavior cache.
	behaviorCache.preservesExecutability = make(map[uint64]bool)
	behaviorCache.decomposesUnicode = make(map[uint64]bool)
}

// scanner provides the recursive implementation of scanning.
type scanner struct {
	// cancelled is the cancellation channel from the scan context.
	cancelled <-chan struct{}
	// root is the path to the synchronization root.
	root string
	// dirtyPaths is the set of tainted paths for which a baseline snapshot
	// can't be trusted.
	dirtyPaths map[string]bool
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
	// copyBuffer is the copy buffer used for computing file digests.
	copyBuffer []byte
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

// file performs processing of a file entry. Exactly one of parent or file will
// be non-nil, depending on whether or not the path represents the
// synchronization root. If the path represents the synchronization root, then
// file will be provided and the caller will be responsible for its closure
// (i.e. this function should not close it). Otherwise, the parent of the path
// is provided and this function is responsible for opening and closing the file
// as necessary.
func (s *scanner) file(
	path string,
	parent *filesystem.Directory,
	metadata *filesystem.Metadata,
	file filesystem.ReadableFile,
) (*Entry, error) {
	// Compute executability.
	executable := s.preservesExecutability && anyExecutableBitSet(metadata.Mode)

	// Try to find cached data for this path.
	cached, cacheHit := s.cache.Entries[path]

	// Check if we can reuse the cached digest (in order to avoid recomputation)
	// and the cache entry itself (in order to avoid allocation). In order for
	// the cached digest to be considered valid, we require that type,
	// modification time, file size, and file ID haven't changed. We don't check
	// for permission bit changes when assessing digest reusability since they
	// don't affect content, but we do check for full mode equivalence when
	// assessing cache entry reusability since permission changes need to be
	// detected during transition operations (where the cache is also used).
	cacheContentMatch := cacheHit &&
		(metadata.Mode&filesystem.ModeTypeMask) == (filesystem.Mode(cached.Mode)&filesystem.ModeTypeMask) &&
		metadata.ModificationTime.Equal(cached.ModificationTime.AsTime()) &&
		metadata.Size == cached.Size &&
		metadata.FileID == cached.FileID
	cacheEntryReusable := cacheContentMatch &&
		metadata.Mode == filesystem.Mode(cached.Mode)

	// Compute the digest, either by pulling it from the cache or computing it
	// from the on-disk contents.
	var digest []byte
	var err error
	if cacheContentMatch {
		digest = cached.Digest
	} else {
		// Open the file if it's not open already. If we do open it, then defer
		// its closure.
		if file == nil {
			file, err = parent.OpenFile(metadata.Name)
			if err != nil {
				return nil, fmt.Errorf("unable to open file (%s): %w", path, err)
			}
			defer file.Close()
		}

		// Reset the hash state.
		s.hasher.Reset()

		// Copy data into the hash and verify that we copied the amount
		// expected. We use a preemptable wrapper around the hasher to enable
		// timely cancellation.
		preemptableHasher := &preemptableWriter{
			cancelled:     s.cancelled,
			writer:        s.hasher,
			checkInterval: scannerCopyPreemptionInterval,
		}
		if copied, err := io.CopyBuffer(preemptableHasher, file, s.copyBuffer); err != nil {
			if err == errWritePreempted {
				return nil, errScanCancelled
			}
			return nil, fmt.Errorf("unable to hash file contents (%s): %w", path, err)
		} else if uint64(copied) != metadata.Size {
			return nil, fmt.Errorf("hashed size mismatch (%s): %d != %d", path, copied, metadata.Size)
		}

		// Compute the digest.
		digest = s.hasher.Sum(nil)
	}

	// Add an entry to the new cache.
	if cacheEntryReusable {
		s.newCache.Entries[path] = cached
	} else {
		// Convert the new modification time to Protocol Buffers format.
		modificationTime := timestamppb.New(metadata.ModificationTime)
		if err := modificationTime.CheckValid(); err != nil {
			return nil, fmt.Errorf("unable to convert file modification time (%s): %w", path, err)
		}

		// Create the new cache entry.
		s.newCache.Entries[path] = &CacheEntry{
			Mode:             uint32(metadata.Mode),
			ModificationTime: modificationTime,
			Size:             metadata.Size,
			FileID:           metadata.FileID,
			Digest:           digest,
		}
	}

	// Success.
	return &Entry{
		Kind:       EntryKind_File,
		Executable: executable,
		Digest:     digest,
	}, nil
}

// symbolicLink performs processing of a symbolic link entry.
func (s *scanner) symbolicLink(
	path string,
	parent *filesystem.Directory,
	name string,
	enforcePortable bool,
) (*Entry, error) {
	// Read the link target.
	target, err := parent.ReadSymbolicLink(name)
	if err != nil {
		return nil, fmt.Errorf("unable to read symbolic link target (%s): %w", path, err)
	}

	// If requested, enforce that the link is portable, otherwise just ensure
	// that it's non-empty (this is required even in POSIX raw mode).
	if enforcePortable {
		target, err = normalizeSymlinkAndEnsurePortable(path, target)
		if err != nil {
			return nil, fmt.Errorf("invalid symbolic link (%s): %w", path, err)
		}
	} else if target == "" {
		return nil, fmt.Errorf("symbolic link target is empty (%s)", path)
	}

	// Success.
	return &Entry{
		Kind:   EntryKind_Symlink,
		Target: target,
	}, nil
}

// directory performs processing of a directory entry. Exactly one of parent or
// directory will be non-nil, depending on whether or not the path represents
// the synchronization root. If the path represents the synchronization root,
// then directory will be provided and the caller will be responsible for its
// closure (i.e. this function should not close it). Otherwise, the parent of
// the path is provided and this function is responsible for opening and closing
// the directory as necessary.
func (s *scanner) directory(
	path string,
	parent *filesystem.Directory,
	metadata *filesystem.Metadata,
	directory *filesystem.Directory,
	baseline *Entry,
) (*Entry, error) {
	// Verify that the baseline, if any, is sane.
	if baseline != nil && baseline.Kind != EntryKind_Directory {
		panic("non-directory baseline passed to directory handler")
	}

	// Verify that we haven't crossed a directory boundary (which might
	// potentially change executability preservation or Unicode decomposition
	// behavior).
	if metadata.DeviceID != s.deviceID {
		return nil, fmt.Errorf("scan crossed filesystem boundary (%s)", path)
	}

	// If the directory is not yet opened, then open it and defer its closure.
	if directory == nil {
		if d, err := parent.OpenDirectory(metadata.Name); err != nil {
			return nil, fmt.Errorf("unable to open directory (%s): %w", path, err)
		} else {
			directory = d
			defer directory.Close()
		}
	}

	// Read directory contents.
	directoryContents, err := directory.ReadContents()
	if err != nil {
		return nil, fmt.Errorf("unable to read directory contents (%s): %w", path, err)
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
	for _, contentMetadata := range directoryContents {
		// Check for cancellation.
		select {
		case <-s.cancelled:
			return nil, errScanCancelled
		default:
		}

		// Extract the content name.
		contentName := contentMetadata.Name

		// If this is an intermediate temporary file, then ignore it.
		if filesystem.IsTemporaryFileName(contentName) {
			continue
		}

		// Recompose Unicode in the content name if necessary.
		if s.recomposeUnicode {
			contentName = norm.NFC.String(contentName)
		}

		// Compute the content path.
		contentPath := pathJoin(path, contentName)

		// Compute the kind for this content, skipping if unsupported.
		var contentKind EntryKind
		switch contentMetadata.Mode & filesystem.ModeTypeMask {
		case filesystem.ModeTypeDirectory:
			contentKind = EntryKind_Directory
		case filesystem.ModeTypeFile:
			contentKind = EntryKind_File
		case filesystem.ModeTypeSymbolicLink:
			contentKind = EntryKind_Symlink
		default:
			continue
		}

		// Determine whether or not this path is ignored and update the new
		// ignore cache.
		contentIsDirectory := contentKind == EntryKind_Directory
		ignoreCacheKey := IgnoreCacheKey{contentPath, contentIsDirectory}
		ignored, ok := s.ignoreCache[ignoreCacheKey]
		if !ok {
			ignored = s.ignorer.ignored(contentPath, contentIsDirectory)
		}
		s.newIgnoreCache[ignoreCacheKey] = ignored
		if ignored {
			continue
		}

		// If we have a baseline, then check if that baseline has content with
		// the same name and kind as what we see on disk. If so, then we can use
		// that as a baseline for the content.
		var contentBaseline *Entry
		if baseline != nil {
			contentBaseline = baseline.Contents[contentName]
			if contentBaseline != nil && contentBaseline.Kind != contentKind {
				contentBaseline = nil
			}
		}

		// If we have a baseline entry for the content and the content path
		// isn't marked as dirty, then we can just re-use that baseline entry
		// directly.
		if contentBaseline != nil {
			if _, contentDirty := s.dirtyPaths[contentPath]; !contentDirty {
				contents[contentName] = contentBaseline
				continue
			}
		}

		// Handle based on kind.
		var entry *Entry
		var err error
		if contentKind == EntryKind_File {
			entry, err = s.file(contentPath, directory, contentMetadata, nil)
		} else if contentKind == EntryKind_Symlink {
			if s.symlinkMode == SymlinkMode_SymlinkModePortable {
				entry, err = s.symbolicLink(contentPath, directory, contentName, true)
			} else if s.symlinkMode == SymlinkMode_SymlinkModeIgnore {
				continue
			} else if s.symlinkMode == SymlinkMode_SymlinkModePOSIXRaw {
				entry, err = s.symbolicLink(contentPath, directory, contentName, false)
			} else {
				panic("unsupported symlink mode")
			}
		} else if contentKind == EntryKind_Directory {
			entry, err = s.directory(contentPath, directory, contentMetadata, nil, contentBaseline)
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

	// Success.
	return &Entry{
		Kind:     EntryKind_Directory,
		Contents: contents,
	}, nil
}

// Scan provides recursive filesystem scanning facilities for synchronization
// roots.
func Scan(
	ctx context.Context,
	root string,
	baseline *Entry,
	recheckPaths map[string]bool,
	hasher hash.Hash,
	cache *Cache,
	ignores []string,
	ignoreCache IgnoreCache,
	probeMode behavior.ProbeMode,
	symlinkMode SymlinkMode,
) (*Entry, bool, bool, *Cache, IgnoreCache, error) {
	// Verify that the symlink mode is valid for this platform.
	if symlinkMode == SymlinkMode_SymlinkModePOSIXRaw && runtime.GOOS == "windows" {
		return nil, false, false, nil, nil, errors.New("raw POSIX symlinks not supported on Windows")
	}

	// Open the root and defer its closure. We explicitly disallow symbolic
	// links at the root path, though intermediate symbolic links are fine.
	rootObject, metadata, err := filesystem.Open(root, false)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, false, &Cache{}, nil, nil
		} else {
			return nil, false, false, nil, nil, fmt.Errorf("unable to open synchronization root: %w", err)
		}
	}
	defer rootObject.Close()

	// Determine the root kind and extract the underlying object.
	var rootKind EntryKind
	var directoryRoot *filesystem.Directory
	var fileRoot filesystem.ReadableFile
	switch metadata.Mode & filesystem.ModeTypeMask {
	case filesystem.ModeTypeDirectory:
		rootKind = EntryKind_Directory
		if d, ok := rootObject.(*filesystem.Directory); !ok {
			panic("invalid directory object returned from root open operation")
		} else {
			directoryRoot = d
		}
	case filesystem.ModeTypeFile:
		rootKind = EntryKind_File
		if f, ok := rootObject.(filesystem.ReadableFile); !ok {
			panic("invalid file object returned from root open operation")
		} else {
			fileRoot = f
		}
	default:
		panic("invalid filesystem type returned from root open operation")
	}

	// Probe the behavior of the synchronization root.
	var decomposesUnicode, preservesExecutability bool
	if rootKind == EntryKind_Directory {
		// Check if there is cached behavior information. This is an indication
		// that we previously had to use probe files for this filesystem.
		behaviorCache.RLock()
		cachedDecomposes, cachedDecomposesOk := behaviorCache.decomposesUnicode[metadata.DeviceID]
		cachedPreserves, cachedPreservesOk := behaviorCache.preservesExecutability[metadata.DeviceID]
		behaviorCache.RUnlock()

		// Track whether or not we use probe files.
		var usedProbeFiles bool

		// Determine Unicode decomposition behavior.
		if cachedDecomposesOk {
			decomposesUnicode = cachedDecomposes
		} else if decomposes, usedFiles, err := behavior.DecomposesUnicode(directoryRoot, probeMode); err != nil {
			return nil, false, false, nil, nil, fmt.Errorf("unable to probe root Unicode decomposition behavior: %w", err)
		} else {
			decomposesUnicode = decomposes
			usedProbeFiles = usedProbeFiles || usedFiles
		}

		// Determine executability preservation behavior.
		if cachedPreservesOk {
			preservesExecutability = cachedPreserves
		} else if preserves, usedFiles, err := behavior.PreservesExecutability(directoryRoot, probeMode); err != nil {
			return nil, false, false, nil, nil, fmt.Errorf("unable to probe root executability preservation behavior: %w", err)
		} else {
			preservesExecutability = preserves
			usedProbeFiles = usedProbeFiles || usedFiles
		}

		// If we used probe files, update the cache.
		if usedProbeFiles {
			behaviorCache.Lock()
			behaviorCache.decomposesUnicode[metadata.DeviceID] = decomposesUnicode
			behaviorCache.preservesExecutability[metadata.DeviceID] = preservesExecutability
			behaviorCache.Unlock()
		}
	} else if rootKind == EntryKind_File {
		// Check if there is cached behavior information. This is an indication
		// that we previously had to use probe files for this filesystem.
		behaviorCache.RLock()
		cachedPreserves, cachedPreservesOk := behaviorCache.preservesExecutability[metadata.DeviceID]
		behaviorCache.RUnlock()

		// Track whether or not we use probe files.
		var usedProbeFiles bool

		// Determine executability preservation behavior for the parent of the
		// root path.
		//
		// RACE: There is technically a race condition here on POSIX systems
		// because the root file that we have open may have been unlinked and
		// the parent directory path removed or replaced. Even if the file
		// hasn't been unlinked, we still have to make this probe by path since
		// there's no way (due to both APIs and underlying designs) to grab a
		// parent directory by file descriptor on POSIX (it's not a well-defined
		// concept (due at least to the existence of hard links)). In any case,
		// the minimal cross-section for this occurrence combined with the minor
		// consequences of such a case arising mean that we're content to live
		// with this situation for now.
		//
		// TODO: Now that we have fstatfs-based behavior checks (which will also
		// work for file roots), we should try to extract behavior information
		// from the file itself before falling back to path-based checks on the
		// parent directory. The only case where we'd need to fall back would be
		// when probe files are used because of an unknown filesystem. In theory
		// we could even fold all of this logic (including the parent path
		// fallback) into the behavior package itself, though it'll be complex
		// because of platform-specific interfaces and the fact that we'd need
		// to pass through the full parent path.
		if cachedPreservesOk {
			preservesExecutability = cachedPreserves
		} else if preserves, usedFiles, err := behavior.PreservesExecutabilityByPath(filepath.Dir(root), probeMode); err != nil {
			return nil, false, false, nil, nil, fmt.Errorf("unable to probe root parent executability preservation behavior: %w", err)
		} else {
			preservesExecutability = preserves
			usedProbeFiles = usedProbeFiles || usedFiles
		}

		// If we used probe files, update the behavior cache. We're okay to
		// cache by device ID here (even though it's the file's device ID) since
		// we know the parent directory will have the same device ID as the file
		// within it.
		if usedProbeFiles {
			behaviorCache.Lock()
			behaviorCache.preservesExecutability[metadata.DeviceID] = preservesExecutability
			behaviorCache.Unlock()
		}
	} else {
		panic("unhandled root kind")
	}

	// If a baseline has been provided but its kind doesn't match that of the
	// synchronization root, then we can ignore it.
	if baseline != nil && baseline.Kind != rootKind {
		baseline = nil
	}

	// If a baseline of the correct kind is available, and there aren't any
	// re-check paths specified, then we can just re-use that baseline directly.
	// We don't explicitly check here that the digest cache and ignore cache
	// correspond to the baseline, because doing so is expensive. We place the
	// burden of enforcing that invariant on the caller.
	if baseline != nil && len(recheckPaths) == 0 {
		return baseline, preservesExecutability, decomposesUnicode, cache, ignoreCache, nil
	}

	// Convert the list of re-check paths into a set of dirty paths. The rule is
	// that we add any re-check path as well as any parent component of any
	// re-check path.
	var dirtyPaths map[string]bool
	if baseline != nil && len(recheckPaths) > 0 {
		dirtyPaths = make(map[string]bool)
		for path := range recheckPaths {
			for {
				dirtyPaths[path] = true
				if path == "" {
					break
				}
				path = pathDir(path)
			}
		}
	}

	// If a nil cache has been provided, convert it to an empty but non-nil
	// version to avoid needing to use the GetEntries accessor everywhere.
	if cache == nil {
		cache = &Cache{}
	}

	// Create the ignorer.
	ignorer, err := newIgnorer(ignores)
	if err != nil {
		return nil, false, false, nil, nil, fmt.Errorf("unable to create ignorer: %w", err)
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
		cancelled:              ctx.Done(),
		root:                   root,
		dirtyPaths:             dirtyPaths,
		hasher:                 hasher,
		cache:                  cache,
		ignorer:                ignorer,
		ignoreCache:            ignoreCache,
		symlinkMode:            symlinkMode,
		newCache:               newCache,
		newIgnoreCache:         newIgnoreCache,
		copyBuffer:             make([]byte, scannerCopyBufferSize),
		deviceID:               metadata.DeviceID,
		recomposeUnicode:       decomposesUnicode,
		preservesExecutability: preservesExecutability,
	}

	// Handle the scan based on the root type.
	var result *Entry
	if rootKind == EntryKind_Directory {
		result, err = s.directory("", nil, metadata, directoryRoot, baseline)
	} else if rootKind == EntryKind_File {
		result, err = s.file("", nil, metadata, fileRoot)
	} else {
		panic("unhandled root kind")
	}
	if err != nil {
		return nil, false, false, nil, nil, err
	}

	// If we have a baseline, then backfill the ignore and digest caches to
	// include entries for paths that exist in our result but which we may not
	// have explicitly visited.
	//
	// In the case of the ignore cache, we just add false entries for each path
	// that we see in the result since (a) these are the only paths that we'd be
	// able to propagate from the old ignore cache anyway and (b) we know their
	// ignore cache value would be false (since they aren't ignored). Obviously
	// we miss out on any true entries in the ignore cache (from previously
	// ignored content), but this is generally fine because (a) the bulk of the
	// ignore cache is non-ignored content anyway (because most ignored content
	// is ignored as the result of a single parent path) and (b) these single
	// missing paths will be cheap enough to re-process later.
	//
	// In the case of the digest cache, we have to ensure correct propagation
	// from the old cache to the new in the case of entries that we didn't
	// explicitly revisit, which we can do because we have the paths for all
	// cache entries which need to be propagated (i.e. those in the result but
	// not in the new cache). We don't perform total validation that the old
	// digest cache corresponds to the baseline (e.g. we don't check digest
	// value matches) because it's too expensive, though we do detect the case
	// of missing entries since it's relatively cheap. The burden of ensuring
	// cache/baseline correspondence technically falls on the caller.
	if baseline != nil {
		// Track missing cache entries.
		var missingCacheEntries bool

		// Perform propagation.
		result.walk("", func(path string, entry *Entry) {
			// Create an ignore cache entry for this path.
			newIgnoreCache[IgnoreCacheKey{path, entry.Kind == EntryKind_Directory}] = false

			// Propagate digest cache entries.
			if entry.Kind == EntryKind_File {
				if _, ok := newCache.Entries[path]; !ok {
					if oldCacheEntry, ok := cache.Entries[path]; ok {
						newCache.Entries[path] = oldCacheEntry
					} else {
						missingCacheEntries = true
					}
				}
			}
		})

		// Abort if we encountered missing cache entries.
		if missingCacheEntries {
			return nil, false, false, nil, nil, errors.New("old cache entries don't correspond to baseline")
		}
	}

	// Success.
	return result, preservesExecutability, decomposesUnicode, newCache, newIgnoreCache, nil
}
