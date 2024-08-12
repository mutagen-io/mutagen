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
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/filesystem/behavior"
	"github.com/mutagen-io/mutagen/pkg/stream"
	"github.com/mutagen-io/mutagen/pkg/synchronization/core/fastpath"
	"github.com/mutagen-io/mutagen/pkg/synchronization/core/ignore"
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

// ErrScanCancelled indicates that the scan was cancelled.
var ErrScanCancelled = errors.New("scan cancelled")

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
	ignorer ignore.Ignorer
	// ignoreCache is the cache of ignored path behavior.
	ignoreCache ignore.IgnoreCache
	// symbolicLinkMode is the symbolic link mode being used.
	symbolicLinkMode SymbolicLinkMode
	// permissionsMode is the permissions mode being used.
	permissionsMode PermissionsMode
	// newCache is the new file digest cache to populate.
	newCache *Cache
	// newIgnoreCache is the new ignored path behavior cache to populate.
	newIgnoreCache ignore.IgnoreCache
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
	// directories is the number of synchronizable directories encountered.
	directories uint64
	// files is the number of synchronizable files encountered.
	files uint64
	// symbolicLinks is the number of synchronizable symbolic links encountered.
	symbolicLinks uint64
	// totalFileSize is the total size of all synchronizable files encountered.
	totalFileSize uint64
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
	file io.ReadSeekCloser,
) (*Entry, error) {
	// Compute executability.
	var executable bool
	if s.permissionsMode == PermissionsMode_PermissionsModePortable {
		executable = s.preservesExecutability && anyExecutableBitSet(metadata.Mode)
	}

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
		// If the file is not yet opened, then open it and defer its closure. We
		// can also update the metadata at this point since we'll pay the cost
		// of accessing it when opening the file.
		if file == nil {
			file, metadata, err = parent.OpenFile(metadata.Name)
			if err != nil {
				if os.IsNotExist(err) {
					return nil, err
				}
				return &Entry{
					Kind:    EntryKind_Problematic,
					Problem: fmt.Errorf("unable to open file: %w", err).Error(),
				}, nil
			}
			defer file.Close()
		}

		// Reset the hash state.
		s.hasher.Reset()

		// Copy data into the hash and verify that we copied the amount
		// expected. We use a preemptable wrapper around the hasher to enable
		// timely cancellation.
		preemptableHasher := stream.NewPreemptableWriter(s.hasher, s.cancelled, scannerCopyPreemptionInterval)
		if copied, err := io.CopyBuffer(preemptableHasher, file, s.copyBuffer); err != nil {
			if errors.Is(err, stream.ErrWritePreempted) {
				return nil, ErrScanCancelled
			}
			return &Entry{
				Kind:    EntryKind_Problematic,
				Problem: fmt.Errorf("unable to hash file contents: %w", err).Error(),
			}, nil
		} else if uint64(copied) != metadata.Size {
			return &Entry{
				Kind:    EntryKind_Problematic,
				Problem: fmt.Sprintf("hashed size mismatch: %d != %d", copied, metadata.Size),
			}, nil
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
			return &Entry{
				Kind:    EntryKind_Problematic,
				Problem: fmt.Errorf("unable to convert file modification time: %w", err).Error(),
			}, nil
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

	// Increment the total file count and size.
	s.files++
	s.totalFileSize += metadata.Size

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
		if os.IsNotExist(err) {
			return nil, err
		}
		return &Entry{
			Kind:    EntryKind_Problematic,
			Problem: fmt.Errorf("unable to read symbolic link target: %w", err).Error(),
		}, nil
	}

	// If requested, enforce that the link is portable, otherwise just ensure
	// that it's non-empty (this is required even in POSIX raw mode).
	if enforcePortable {
		target, err = normalizeSymbolicLinkAndEnsurePortable(path, target)
		if err != nil {
			return &Entry{
				Kind:    EntryKind_Problematic,
				Problem: fmt.Errorf("invalid symbolic link: %w", err).Error(),
			}, nil
		}
	} else if target == "" {
		return &Entry{
			Kind:    EntryKind_Problematic,
			Problem: "symbolic link target is empty",
		}, nil
	}

	// Increment the total symbolic link count.
	s.symbolicLinks++

	// Success.
	return &Entry{
		Kind:   EntryKind_SymbolicLink,
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
	ignoreMask bool,
) (*Entry, error) {
	// Verify that the baseline, if any, is sane.
	if baseline != nil && baseline.Kind != EntryKind_Directory {
		panic("non-directory baseline passed to directory handler")
	}

	// Verify that we haven't crossed a directory boundary (which might
	// potentially change executability preservation or Unicode decomposition
	// behavior).
	if metadata.DeviceID != s.deviceID {
		return &Entry{
			Kind:    EntryKind_Problematic,
			Problem: "scan crossed filesystem boundary",
		}, nil
	}

	// If the directory is not yet opened, then open it and defer its closure.
	if directory == nil {
		if d, err := parent.OpenDirectory(metadata.Name); err != nil {
			if os.IsNotExist(err) {
				return nil, err
			}
			return &Entry{
				Kind:    EntryKind_Problematic,
				Problem: fmt.Errorf("unable to open directory: %w", err).Error(),
			}, nil
		} else {
			directory = d
			defer directory.Close()
		}
	}

	// Read directory contents.
	directoryContents, err := directory.ReadContents()
	if err != nil {
		return &Entry{
			Kind:    EntryKind_Problematic,
			Problem: fmt.Errorf("unable to read directory contents: %w", err).Error(),
		}, nil
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

	// Compute the prefix to add to content names to compute their paths.
	var contentPathPrefix string
	if len(directoryContents) > 0 {
		contentPathPrefix = fastpath.Joinable(path)
	}

	// Compute the entries for the content map.
	contents := make(map[string]*Entry, len(directoryContents))
	for _, contentMetadata := range directoryContents {
		// Check for cancellation.
		select {
		case <-s.cancelled:
			return nil, ErrScanCancelled
		default:
		}

		// Extract the content name.
		contentName := contentMetadata.Name

		// If this is an intermediate temporary file, then ignore it. We avoid
		// recording these files, even as untracked entries, because we know
		// that they're ephemeral.
		if strings.HasPrefix(contentName, filesystem.TemporaryNamePrefix) {
			continue
		}

		// If the filename is not valid UTF-8, then flag it as either untracked
		// or problematic content, depending on the ignore mask. The reason for
		// this distinction is that all non-UTF-8-named content inherently falls
		// under IgnoreStatusNominal (because the name couldn't possibly match
		// any ignore (or unignore) specification). Moreover, we wouldn't want a
		// non-UTF-8-named entry to be the sole trigger that reified a phantom
		// directory into existence, and even if other content were to trigger a
		// phantom directory into existence, we wouldn't care about the
		// non-UTF-8-named entry because it would be ignore masked out.
		//
		// UTF-8 enforcement is important for both (a) ensuring that comparisons
		// are performed using a common encoding and (b) allowing the name to be
		// encoded with Protocol Buffers (which enforces that strings are UTF-8
		// encoded when marshaling). Since the file name isn't valid for storing
		// in the content map, we'll replace all unknown byte sequences with a
		// replacement character and store the entry with a (hopefully)
		// non-coliding derivative name.
		if !utf8.ValidString(contentName) {
			escapedContentName := strings.ToValidUTF8(contentName, "�") + " (non-UTF-8)"
			if ignoreMask {
				contents[escapedContentName] = &Entry{Kind: EntryKind_Untracked}
			} else {
				contents[escapedContentName] = &Entry{
					Kind:    EntryKind_Problematic,
					Problem: "non-UTF-8 filename",
				}
			}
			continue
		}

		// Recompose Unicode in the content name if necessary.
		if s.recomposeUnicode {
			contentName = norm.NFC.String(contentName)
		}

		// Compute the content path.
		contentPath := contentPathPrefix + contentName

		// Compute the kind for this content, recording an untracked entry if
		// the content type isn't supported.
		var contentKind EntryKind
		switch contentMetadata.Mode & filesystem.ModeTypeMask {
		case filesystem.ModeTypeDirectory:
			contentKind = EntryKind_Directory
		case filesystem.ModeTypeFile:
			contentKind = EntryKind_File
		case filesystem.ModeTypeSymbolicLink:
			contentKind = EntryKind_SymbolicLink
		default:
			contents[contentName] = &Entry{Kind: EntryKind_Untracked}
			continue
		}

		// Determine whether or not this path is ignored and update the new
		// ignore cache. If the path is ignored, then record an untracked entry.
		contentIsDirectory := contentKind == EntryKind_Directory
		ignoreCacheKey := ignore.IgnoreCacheKey{contentPath, contentIsDirectory}
		ignoreBehavior, ok := s.ignoreCache[ignoreCacheKey]
		if !ok {
			ignoreStatus, continueTraversal := s.ignorer.Ignore(contentPath, contentIsDirectory)
			ignoreBehavior = ignore.IgnoreCacheValue{ignoreStatus, continueTraversal}
		}
		s.newIgnoreCache[ignoreCacheKey] = ignoreBehavior
		contentIgnoreMask := ignoreMask
		if ignoreBehavior.Status == ignore.IgnoreStatusNominal {
			if ignoreMask && !ignoreBehavior.ContinueTraversal {
				contents[contentName] = &Entry{Kind: EntryKind_Untracked}
				continue
			}
		} else if ignoreBehavior.Status == ignore.IgnoreStatusIgnored {
			if !ignoreBehavior.ContinueTraversal {
				contents[contentName] = &Entry{Kind: EntryKind_Untracked}
				continue
			}
			contentIgnoreMask = true
		} else if ignoreBehavior.Status == ignore.IgnoreStatusUnignored {
			contentIgnoreMask = false
		} else {
			panic("unhandled ignore status")
		}

		// If this is a directory, and we have a baseline, then check if that
		// baseline has content with the same name that is also a directory. If
		// so, then we can use that as a baseline for this content. While we
		// could do this for all entry types, we restrict this optimization to
		// directories because they're the only content types for which
		// rescanning is not O(1). Moreover, we've already paid the price to
		// grab file metadata, so we may as well compare it with the cache since
		// that doesn't require another trip to disk. It's true that we are
		// incurring additional symbolic link reads that we could potentially
		// replace with baseline content, but they are statistically rarer, they
		// only require a single system call, and we're only performing them in
		// directories marked as dirty, so the additional cost is very low.
		var directoryBaseline *Entry
		if contentIsDirectory && baseline != nil {
			directoryBaseline = baseline.Contents[contentName]
			if directoryBaseline != nil && directoryBaseline.Kind != EntryKind_Directory {
				directoryBaseline = nil
			}
		}

		// If we have a baseline entry for the content and the content path
		// isn't marked as dirty, then we can just re-use that baseline entry
		// directly. In this case, we'll want to walk down the entry and
		// propagate the corresponding ignore and digest cache entries that
		// we're going to avoid generating.
		//
		// On Linux, there's one additional heuristic used here: if the content
		// path is not marked as dirty, but the baseline is a directory with no
		// contents and the parent path is marked as dirty (which we can assume
		// here implicitly), then we mark the content as dirty. The reason for
		// this is that fanotify only sees the parent path for operations where
		// a filesystem entry is created, deleted, or renamed into place. For
		// files and symbolic links, we always perform a recheck anyway (since
		// we've already paid for their metadata), but for directories, there's
		// one corner case that has to be handled: the case where an empty
		// directory is deleted and then a different directory with the same
		// name and non-zero contents is renamed into its place. In this case,
		// no events would be generated for the deleted directory (since it
		// would have no content to remove) or the renamed directory (since the
		// operation is atomic and only affects the parent directory) and thus
		// only the parent path would be seen, with no indication that the child
		// directory had been modified. FSEvents and ReadDirectoryChangesW don't
		// have this problem, because they would report the path for the
		// directory being deleted and/or renamed.
		if directoryBaseline != nil {
			contentDirty := s.dirtyPaths[contentPath]
			if runtime.GOOS == "linux" && !contentDirty &&
				len(directoryBaseline.Contents) == 0 {
				contentDirty = true
			}
			if !contentDirty {
				contents[contentName] = directoryBaseline
				var missingCacheEntries bool
				directoryBaseline.walk(contentPath, func(path string, entry *Entry) {
					// Update total entry counts.
					entryIsDirectoryKind := entry.Kind == EntryKind_Directory ||
						entry.Kind == EntryKind_PhantomDirectory
					if entryIsDirectoryKind {
						s.directories++
					} else if entry.Kind == EntryKind_File {
						s.files++
					} else if entry.Kind == EntryKind_SymbolicLink {
						s.symbolicLinks++
					}

					// Propagate any ignore cache entries that we can.
					if entry.Kind != EntryKind_Untracked && entry.Kind != EntryKind_Problematic {
						ignoreCacheKey := ignore.IgnoreCacheKey{path, entryIsDirectoryKind}
						if ignoreBehavior, ok := s.ignoreCache[ignoreCacheKey]; ok {
							s.newIgnoreCache[ignoreCacheKey] = ignoreBehavior
						}
					}

					// Propagate digest cache entries and update total file
					// size. Here we require exhaustive propagation to verify
					// that the baseline corresponds to the provided cache,
					// though note that this is not a full verification (e.g. we
					// don't check that digests or modes match) because that
					// would be too costly.
					if entry.Kind == EntryKind_File {
						if oldCacheEntry, ok := s.cache.Entries[path]; ok {
							s.newCache.Entries[path] = oldCacheEntry
							s.totalFileSize += oldCacheEntry.Size
						} else {
							missingCacheEntries = true
						}
					}
				}, false)
				if missingCacheEntries {
					return nil, errors.New("old cache entries don't correspond to baseline")
				}
				continue
			}
		}

		// If we didn't have a baseline, or if the content path was marked as
		// dirty, then we need to handle it manually. Note that we're still
		// passing the directory baseline down at this point, because its child
		// entries may not be marked as dirty and may be reusable.
		var entry *Entry
		var err error
		if contentKind == EntryKind_File {
			entry, err = s.file(contentPath, directory, contentMetadata, nil)
		} else if contentKind == EntryKind_SymbolicLink {
			if s.symbolicLinkMode == SymbolicLinkMode_SymbolicLinkModePortable {
				entry, err = s.symbolicLink(contentPath, directory, contentName, true)
			} else if s.symbolicLinkMode == SymbolicLinkMode_SymbolicLinkModeIgnore {
				entry = &Entry{Kind: EntryKind_Untracked}
			} else if s.symbolicLinkMode == SymbolicLinkMode_SymbolicLinkModePOSIXRaw {
				entry, err = s.symbolicLink(contentPath, directory, contentName, false)
			} else {
				panic("unsupported symbolic link mode")
			}
		} else if contentKind == EntryKind_Directory {
			entry, err = s.directory(
				contentPath,
				directory, contentMetadata, nil,
				directoryBaseline,
				contentIgnoreMask,
			)
		} else {
			panic("unhandled entry kind")
		}

		// Watch for errors from the handling function. If the error is due to
		// the content no longer existing, then we just treat the content as if
		// it had never existed.
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		// Record the content.
		contents[contentName] = entry
	}

	// Determine the kind of directory that we'll yield. We could do more
	// aggressive reification of phantom directories here (converting them to
	// tracked if we already know that they contain trackable content), but
	// given that we still have to walk snapshots in their entirety during
	// reification (to handle cases where we don't know for sure either way
	// until we have both snapshots), it doesn't make sense to add the
	// additional complexity of managing that tracking during scanning, nor
	// would it save us any transfer size (since we'd only be able to reify to
	// tracked here, not ignored).
	directoryKind := EntryKind_Directory
	if ignoreMask {
		directoryKind = EntryKind_PhantomDirectory
	}

	// Increment the total directory count. We still include phantom directories
	// in this count since we're paying the cost of transmitting them. The count
	// will be updated during reification, if necessary.
	s.directories++

	// Success.
	return &Entry{
		Kind:     directoryKind,
		Contents: contents,
	}, nil
}

// Scan creates a new filesystem snapshot at the specified root. The only
// required arguments are ctx, root, hasher, ignores, probeMode,
// symbolicLinkMode, and permissionsMode. The baseline, recheckPaths, cache, and
// ignoreCache fields merely provide acceleration options.
func Scan(
	ctx context.Context,
	root string,
	baseline *Snapshot, recheckPaths map[string]bool,
	hasher hash.Hash, cache *Cache,
	ignorer ignore.Ignorer, ignoreCache ignore.IgnoreCache,
	probeMode behavior.ProbeMode,
	symbolicLinkMode SymbolicLinkMode,
	permissionsMode PermissionsMode,
) (*Snapshot, *Cache, ignore.IgnoreCache, error) {
	// Verify that the symbolic link mode is valid for this platform.
	if symbolicLinkMode == SymbolicLinkMode_SymbolicLinkModePOSIXRaw && runtime.GOOS == "windows" {
		return nil, nil, nil, errors.New("raw POSIX symbolic links not supported on Windows")
	}

	// Open the root and defer its closure. We explicitly disallow symbolic
	// links at the root path, though intermediate symbolic links are fine.
	rootObject, metadata, err := filesystem.Open(root, false)
	if err != nil {
		if os.IsNotExist(err) {
			return &Snapshot{}, &Cache{}, nil, nil
		} else {
			return nil, nil, nil, fmt.Errorf("unable to open synchronization root: %w", err)
		}
	}
	defer rootObject.Close()

	// Determine the root kind and extract the underlying object.
	var rootKind EntryKind
	var directoryRoot *filesystem.Directory
	var fileRoot io.ReadSeekCloser
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
		if f, ok := rootObject.(io.ReadSeekCloser); !ok {
			panic("invalid file object returned from root open operation")
		} else {
			fileRoot = f
		}
	default:
		panic("invalid filesystem type returned from root open operation")
	}

	// Check if there is cached behavior information.
	behaviorCache.RLock()
	cachedPreserves, cachedPreservesOk := behaviorCache.preservesExecutability[metadata.DeviceID]
	cachedDecomposes, cachedDecomposesOk := behaviorCache.decomposesUnicode[metadata.DeviceID]
	behaviorCache.RUnlock()

	// Track whether or not we use probe files when determining behavior.
	var usedProbeFiles bool

	// Probe the behavior of the synchronization root.
	var preservesExecutability, decomposesUnicode bool
	if rootKind == EntryKind_Directory {
		// Check executability preservation behavior.
		if cachedPreservesOk {
			preservesExecutability = cachedPreserves
		} else if preserves, usedFiles, err := behavior.PreservesExecutability(directoryRoot, probeMode); err != nil {
			return nil, nil, nil, fmt.Errorf("unable to probe root executability preservation behavior: %w", err)
		} else {
			preservesExecutability = preserves
			usedProbeFiles = usedProbeFiles || usedFiles
		}

		// Check Unicode decomposition behavior.
		if cachedDecomposesOk {
			decomposesUnicode = cachedDecomposes
		} else if decomposes, usedFiles, err := behavior.DecomposesUnicode(directoryRoot, probeMode); err != nil {
			return nil, nil, nil, fmt.Errorf("unable to probe root Unicode decomposition behavior: %w", err)
		} else {
			decomposesUnicode = decomposes
			usedProbeFiles = usedProbeFiles || usedFiles
		}
	} else if rootKind == EntryKind_File {
		// For file roots, we use the behavioral information of their parent
		// directory.
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
		// with this situation for now. Note, however, that this could affect
		// the behavior caches for other sessions as well.
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
		parent := filepath.Dir(root)

		// Check executability preservation behavior for the parent directory.
		if cachedPreservesOk {
			preservesExecutability = cachedPreserves
		} else if preserves, usedFiles, err := behavior.PreservesExecutabilityByPath(parent, probeMode); err != nil {
			return nil, nil, nil, fmt.Errorf("unable to probe parent executability preservation behavior: %w", err)
		} else {
			preservesExecutability = preserves
			usedProbeFiles = usedProbeFiles || usedFiles
		}

		// Check Unicode decomposition behavior for the parent directory.
		if cachedDecomposesOk {
			decomposesUnicode = cachedDecomposes
		} else if decomposes, usedFiles, err := behavior.DecomposesUnicodeByPath(parent, probeMode); err != nil {
			return nil, nil, nil, fmt.Errorf("unable to probe parent Unicode decomposition behavior: %w", err)
		} else {
			decomposesUnicode = decomposes
			usedProbeFiles = usedProbeFiles || usedFiles
		}
	} else {
		panic("unhandled root kind")
	}

	// If we used probe files, then update the behavior cache, because probing
	// was relatively expensive. Probe files are never used on Windows, so we're
	// safe to use the device ID (which is always 0 on Windows) as a cache key.
	if usedProbeFiles {
		behaviorCache.Lock()
		behaviorCache.preservesExecutability[metadata.DeviceID] = preservesExecutability
		behaviorCache.decomposesUnicode[metadata.DeviceID] = decomposesUnicode
		behaviorCache.Unlock()
	}

	// If a baseline has been provided but differs in terms of root kind or
	// filesystem behavior, then we can just ignore it.
	if baseline != nil {
		baselineInvalid := baseline.Content == nil ||
			baseline.Content.Kind != rootKind ||
			baseline.PreservesExecutability != preservesExecutability ||
			baseline.DecomposesUnicode != decomposesUnicode
		if baselineInvalid {
			baseline = nil
		}
	}

	// If a baseline of the correct kind is available, and there aren't any
	// re-check paths specified, then we can just re-use that baseline directly.
	// We don't explicitly check here that the digest cache and ignore cache
	// correspond to the baseline, because doing so is expensive. We place the
	// burden of enforcing that invariant on the caller.
	if baseline != nil && len(recheckPaths) == 0 {
		return baseline, cache, ignoreCache, nil
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
				path = fastpath.Dir(path)
			}
		}
	}

	// If a nil cache has been provided, convert it to an empty but non-nil
	// version to avoid needing to use the GetEntries accessor everywhere.
	if cache == nil {
		cache = &Cache{}
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
	newIgnoreCache := make(ignore.IgnoreCache, initialIgnoreCacheCapacity)

	// Create a scanner.
	s := &scanner{
		cancelled:              ctx.Done(),
		root:                   root,
		dirtyPaths:             dirtyPaths,
		hasher:                 hasher,
		cache:                  cache,
		ignorer:                ignorer,
		ignoreCache:            ignoreCache,
		symbolicLinkMode:       symbolicLinkMode,
		permissionsMode:        permissionsMode,
		newCache:               newCache,
		newIgnoreCache:         newIgnoreCache,
		copyBuffer:             make([]byte, scannerCopyBufferSize),
		deviceID:               metadata.DeviceID,
		recomposeUnicode:       decomposesUnicode,
		preservesExecutability: preservesExecutability,
	}

	// Handle the scan based on the root type.
	var content *Entry
	if rootKind == EntryKind_Directory {
		var directoryBaseline *Entry
		if baseline != nil {
			directoryBaseline = baseline.Content
		}
		content, err = s.directory("", nil, metadata, directoryRoot, directoryBaseline, false)
	} else if rootKind == EntryKind_File {
		content, err = s.file("", nil, metadata, fileRoot)
	} else {
		panic("unhandled root kind")
	}
	if err != nil {
		return nil, nil, nil, err
	}

	// Success.
	return &Snapshot{
		Content:                content,
		PreservesExecutability: preservesExecutability,
		DecomposesUnicode:      decomposesUnicode,
		Directories:            s.directories,
		Files:                  s.files,
		SymbolicLinks:          s.symbolicLinks,
		TotalFileSize:          s.totalFileSize,
	}, newCache, newIgnoreCache, nil
}
