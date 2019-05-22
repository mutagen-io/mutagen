package sync

import (
	"crypto/sha1"
	"fmt"
	"hash"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/filesystem/behavior"
)

// testAcceleratedCacheIsSubset verifies that accelerated is a subset of
// original, excluding the presence of a root path key in accelerated.
func testAcceleratedCacheIsSubset(accelerated, original IgnoreCache) bool {
	// Check values.
	for key, value := range accelerated {
		if key.path == "" {
			continue
		} else if other, ok := original[key]; !ok || other != value {
			return false
		}
	}

	// Success.
	return true
}

func testCreateScanCycle(temporaryDirectory string, entry *Entry, contentMap map[string][]byte, ignores []string, symlinkMode SymlinkMode, expectEqual bool) error {
	// Create test content on disk and defer its removal.
	root, parent, err := testTransitionCreate(temporaryDirectory, entry, contentMap, false)
	if err != nil {
		return errors.Wrap(err, "unable to create test content")
	}
	defer os.RemoveAll(parent)

	// Create a hasher.
	hasher := newTestHasher()

	// Perform a scan.
	snapshot, preservesExecutability, decomposesUnicode, cache, ignoreCache, err := Scan(
		root,
		nil, nil,
		hasher, nil,
		ignores, nil,
		behavior.ProbeMode_ProbeModeProbe,
		symlinkMode,
	)
	if !preservesExecutability {
		snapshot = PropagateExecutability(nil, entry, snapshot)
	}
	if err != nil {
		return errors.Wrap(err, "unable to perform scan")
	} else if cache == nil {
		return errors.New("nil cache returned")
	} else if expectEqual && !snapshot.Equal(entry) {
		return errors.New("snapshot not equal to expected")
	} else if !expectEqual && snapshot.Equal(entry) {
		return errors.New("snapshot should not have equaled original")
	}

	// Perform an accelerated scan (with a re-check path) using the snapshot as
	// a baseline.
	newSnapshot, newPreservesExecutability, newDecomposesUnicode, newCache, newIgnoreCache, err := Scan(
		root,
		snapshot, map[string]bool{"fake path": true},
		hasher, cache,
		ignores, ignoreCache,
		behavior.ProbeMode_ProbeModeProbe,
		symlinkMode,
	)
	if !newPreservesExecutability {
		newSnapshot = PropagateExecutability(nil, entry, newSnapshot)
	}
	if err != nil {
		return errors.Wrap(err, "unable to perform accelerated scan (with re-check path)")
	} else if !newSnapshot.Equal(snapshot) {
		return errors.New("accelerated snapshot (with re-check paths) not equal to baseline")
	} else if newPreservesExecutability != preservesExecutability {
		return errors.New(
			"accelerated snapshot (with re-check paths) differed in executability preservation behavior",
		)
	} else if newDecomposesUnicode != decomposesUnicode {
		return errors.New(
			"accelerated snapshot (with re-check paths) differed in Unicode decomposition behavior",
		)
	} else if newCache == nil {
		return errors.New("nil cache returned")
	} else if !newCache.Equal(cache) {
		return errors.New("accelerated cache does not match baseline cache")
	} else if !testAcceleratedCacheIsSubset(newIgnoreCache, ignoreCache) {
		return errors.New("accelerated ignore cache does not match baseline")
	} else if expectEqual && !newSnapshot.Equal(entry) {
		return errors.New("snapshot not equal to expected")
	} else if !expectEqual && newSnapshot.Equal(entry) {
		return errors.New("snapshot should not have equaled original")
	}

	// Perform an accelerated scan (without any re-check paths) using the
	// snapshot as a baseline.
	newSnapshot, newPreservesExecutability, newDecomposesUnicode, newCache, newIgnoreCache, err = Scan(
		root,
		snapshot, nil,
		hasher, cache,
		ignores, ignoreCache,
		behavior.ProbeMode_ProbeModeProbe,
		symlinkMode,
	)
	if !newPreservesExecutability {
		newSnapshot = PropagateExecutability(nil, entry, newSnapshot)
	}
	if err != nil {
		return errors.Wrap(err, "unable to perform accelerated scan (with re-check path)")
	} else if !newSnapshot.Equal(snapshot) {
		return errors.New("accelerated snapshot (with re-check paths) not equal to baseline")
	} else if newPreservesExecutability != preservesExecutability {
		return errors.New(
			"accelerated snapshot (with re-check paths) differed in executability preservation behavior",
		)
	} else if newDecomposesUnicode != decomposesUnicode {
		return errors.New(
			"accelerated snapshot (with re-check paths) differed in Unicode decomposition behavior",
		)
	} else if newCache == nil {
		return errors.New("nil cache returned")
	} else if !newCache.Equal(cache) {
		return errors.New("accelerated cache does not match baseline cache")
	} else if !testAcceleratedCacheIsSubset(newIgnoreCache, ignoreCache) {
		return errors.New("accelerated ignore cache does not match baseline")
	} else if expectEqual && !newSnapshot.Equal(entry) {
		return errors.New("snapshot not equal to expected")
	} else if !expectEqual && newSnapshot.Equal(entry) {
		return errors.New("snapshot should not have equaled original")
	}

	// Success.
	return nil
}

func testCreateScanCycleWithPermutations(entry *Entry, contentMap map[string][]byte, ignores []string, symlinkMode SymlinkMode, expectEqual bool) error {
	// Run the underlying test for each of our temporary directories.
	for _, temporaryDirectory := range testingTemporaryDirectories() {
		if err := testCreateScanCycle(temporaryDirectory.path, entry, contentMap, ignores, symlinkMode, expectEqual); err != nil {
			return errors.Wrap(err, fmt.Sprintf("create/scan cycle failed for %s temporary directory", temporaryDirectory.name))
		}
	}

	// Success.
	return nil
}

func TestScanNilRoot(t *testing.T) {
	if err := testCreateScanCycleWithPermutations(testNilEntry, nil, nil, SymlinkMode_SymlinkModePortable, true); err != nil {
		t.Error("creation/scan cycle failed:", err)
	}
}

func TestScanFile1Root(t *testing.T) {
	if err := testCreateScanCycleWithPermutations(testFile1Entry, testFile1ContentMap, nil, SymlinkMode_SymlinkModePortable, true); err != nil {
		t.Error("creation/scan cycle failed:", err)
	}
}

func TestScanFile2Root(t *testing.T) {
	if err := testCreateScanCycleWithPermutations(testFile2Entry, testFile2ContentMap, nil, SymlinkMode_SymlinkModePortable, true); err != nil {
		t.Error("creation/scan cycle failed:", err)
	}
}

func TestScanFile3Root(t *testing.T) {
	if err := testCreateScanCycleWithPermutations(testFile3Entry, testFile3ContentMap, nil, SymlinkMode_SymlinkModePortable, true); err != nil {
		t.Error("creation/scan cycle failed:", err)
	}
}

func TestScanDirectory1Root(t *testing.T) {
	if err := testCreateScanCycleWithPermutations(testDirectory1Entry, testDirectory1ContentMap, nil, SymlinkMode_SymlinkModePortable, true); err != nil {
		t.Error("creation/scan cycle failed:", err)
	}
}

func TestScanDirectory2Root(t *testing.T) {
	if err := testCreateScanCycleWithPermutations(testDirectory2Entry, testDirectory2ContentMap, nil, SymlinkMode_SymlinkModePortable, true); err != nil {
		t.Error("creation/scan cycle failed:", err)
	}
}

func TestScanDirectory3Root(t *testing.T) {
	if err := testCreateScanCycleWithPermutations(testDirectory3Entry, testDirectory3ContentMap, nil, SymlinkMode_SymlinkModePortable, true); err != nil {
		t.Error("creation/scan cycle failed:", err)
	}
}

func TestScanDirectorySaneSymlinkSane(t *testing.T) {
	if err := testCreateScanCycleWithPermutations(testDirectoryWithSaneSymlink, nil, nil, SymlinkMode_SymlinkModePortable, true); err != nil {
		t.Error("sane symlink not allowed inside root with sane symlink mode:", err)
	}
}

func TestScanDirectorySaneSymlinkIgnore(t *testing.T) {
	if err := testCreateScanCycleWithPermutations(testDirectoryWithSaneSymlink, nil, nil, SymlinkMode_SymlinkModeIgnore, false); err != nil {
		t.Error("sane symlink not allowed inside root with ignore symlink mode:", err)
	}
}

func TestScanDirectorySaneSymlinkPOSIXRaw(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	if err := testCreateScanCycleWithPermutations(testDirectoryWithSaneSymlink, nil, nil, SymlinkMode_SymlinkModePOSIXRaw, true); err != nil {
		t.Error("sane symlink not allowed inside root with POSIX raw symlink mode:", err)
	}
}

func TestScanDirectoryInvalidSymlinkNotSane(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	if testCreateScanCycleWithPermutations(testDirectoryWithInvalidSymlink, nil, nil, SymlinkMode_SymlinkModePortable, true) == nil {
		t.Error("invalid symlink allowed inside root with sane symlink mode")
	}
}

func TestScanDirectoryInvalidSymlinkIgnore(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	if err := testCreateScanCycleWithPermutations(testDirectoryWithInvalidSymlink, nil, nil, SymlinkMode_SymlinkModeIgnore, false); err != nil {
		t.Error("invalid symlink not allowed inside root with ignore symlink mode:", err)
	}
}

func TestScanDirectoryInvalidSymlinkPOSIXRaw(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	if err := testCreateScanCycleWithPermutations(testDirectoryWithInvalidSymlink, nil, nil, SymlinkMode_SymlinkModePOSIXRaw, true); err != nil {
		t.Error("invalid symlink not allowed inside root with POSIX raw symlink mode:", err)
	}
}

func TestScanDirectoryEscapingSymlinkSane(t *testing.T) {
	if testCreateScanCycleWithPermutations(testDirectoryWithEscapingSymlink, nil, nil, SymlinkMode_SymlinkModePortable, true) == nil {
		t.Error("escaping symlink allowed inside root with sane symlink mode")
	}
}

func TestScanDirectoryEscapingSymlinkIgnore(t *testing.T) {
	if err := testCreateScanCycleWithPermutations(testDirectoryWithEscapingSymlink, nil, nil, SymlinkMode_SymlinkModeIgnore, false); err != nil {
		t.Error("escaping symlink not allowed inside root with ignore symlink mode:", err)
	}
}

func TestScanDirectoryEscapingSymlinkPOSIXRaw(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	if err := testCreateScanCycleWithPermutations(testDirectoryWithEscapingSymlink, nil, nil, SymlinkMode_SymlinkModePOSIXRaw, true); err != nil {
		t.Error("escaping symlink not allowed inside root with POSIX raw symlink mode:", err)
	}
}

func TestScanDirectoryAbsoluteSymlinkSane(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	if testCreateScanCycleWithPermutations(testDirectoryWithAbsoluteSymlink, nil, nil, SymlinkMode_SymlinkModePortable, true) == nil {
		t.Error("escaping symlink allowed inside root with sane symlink mode")
	}
}

func TestScanDirectoryAbsoluteSymlinkIgnore(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	if err := testCreateScanCycleWithPermutations(testDirectoryWithAbsoluteSymlink, nil, nil, SymlinkMode_SymlinkModeIgnore, false); err != nil {
		t.Error("escaping symlink not allowed inside root with ignore symlink mode:", err)
	}
}

func TestScanDirectoryAbsoluteSymlinkPOSIXRaw(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	if err := testCreateScanCycleWithPermutations(testDirectoryWithAbsoluteSymlink, nil, nil, SymlinkMode_SymlinkModePOSIXRaw, true); err != nil {
		t.Error("escaping symlink not allowed inside root with POSIX raw symlink mode:", err)
	}
}

func TestScanPOSIXRawNotAllowedOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip()
	}
	if testCreateScanCycleWithPermutations(testDirectoryWithSaneSymlink, nil, nil, SymlinkMode_SymlinkModePOSIXRaw, true) == nil {
		t.Error("POSIX raw symlink mode allowed for scan on Windows")
	}
}

func TestScanInvalidIgnores(t *testing.T) {
	if testCreateScanCycleWithPermutations(testDirectory1Entry, testDirectory1ContentMap, []string{""}, SymlinkMode_SymlinkModePortable, true) == nil {
		t.Error("scan allowed with invalid ignore specification")
	}
}

func TestScanIgnore(t *testing.T) {
	if err := testCreateScanCycleWithPermutations(testDirectory1Entry, testDirectory1ContentMap, []string{"second directory"}, SymlinkMode_SymlinkModePortable, false); err != nil {
		t.Error("unexpected result when ignoring directory:", err)
	}
}

func TestScanIgnoreDirectory(t *testing.T) {
	if err := testCreateScanCycleWithPermutations(testDirectory1Entry, testDirectory1ContentMap, []string{"directory/"}, SymlinkMode_SymlinkModePortable, false); err != nil {
		t.Error("unexpected result when ignoring directory:", err)
	}
}

func TestScanFileNotIgnoredOnDirectorySpecification(t *testing.T) {
	if err := testCreateScanCycleWithPermutations(testDirectory1Entry, testDirectory1ContentMap, []string{"file/"}, SymlinkMode_SymlinkModePortable, true); err != nil {
		t.Error("unexpected result when ignoring directory:", err)
	}
}

func TestScanSubfileNotIgnoredOnRootSpecification(t *testing.T) {
	if err := testCreateScanCycleWithPermutations(testDirectory1Entry, testDirectory1ContentMap, []string{"/subfile.exe"}, SymlinkMode_SymlinkModePortable, true); err != nil {
		t.Error("unexpected result when ignoring directory:", err)
	}
}

func TestScanSymlinkRoot(t *testing.T) {
	// Create a temporary directory and defer its cleanup.
	parent, err := ioutil.TempDir("", "mutagen_simulated")
	if err != nil {
		t.Fatal("unable to create temporary directory:", err)
	}
	defer os.RemoveAll(parent)

	// Compute the symlink path.
	root := filepath.Join(parent, "root")

	// Create a symlink inside the parent.
	if err := os.Symlink("relative", root); err != nil {
		t.Fatal("unable to create symlink:", err)
	}

	// Attempt a scan of the symlink.
	if _, _, _, _, _, err := Scan(
		root,
		nil,
		nil,
		sha1.New(),
		nil,
		nil,
		nil,
		behavior.ProbeMode_ProbeModeProbe,
		SymlinkMode_SymlinkModePortable,
	); err == nil {
		t.Error("scan of symlink root allowed")
	}
}

// rescanHashProxy wraps an instance of and implements hash.Hash, but it signals
// a test error if any hashing occurs. It is a test fixture for
// TestEfficientRescan.
type rescanHashProxy struct {
	hash.Hash
	t *testing.T
}

// Sum implements hash.Hash's Sum method, delegating to the underlying hash, but
// signals an error if invoked.
func (p *rescanHashProxy) Sum(b []byte) []byte {
	p.t.Error("rehashing occurred")
	return p.Hash.Sum(b)
}

func TestEfficientRescan(t *testing.T) {
	// Create test content on disk and defer its removal. We only test on the
	// default temporary directory.
	root, parent, err := testTransitionCreate("", testDirectory1Entry, testDirectory1ContentMap, false)
	if err != nil {
		t.Fatal("unable to create test content on disk:", err)
	}
	defer os.RemoveAll(parent)

	// Create a hasher.
	hasher := newTestHasher()

	// Create an initial snapshot and validate the results.
	snapshot, preservesExecutability, _, cache, _, err := Scan(
		root,
		nil,
		nil,
		hasher,
		nil,
		nil,
		nil,
		behavior.ProbeMode_ProbeModeProbe,
		SymlinkMode_SymlinkModePortable,
	)
	if !preservesExecutability {
		snapshot = PropagateExecutability(nil, testDirectory1Entry, snapshot)
	}
	if err != nil {
		t.Fatal("unable to create snapshot:", err)
	} else if cache == nil {
		t.Fatal("nil cache returned")
	} else if !snapshot.Equal(testDirectory1Entry) {
		t.Error("snapshot did not match expected")
	}

	// Hang on to the old cache.
	oldCache := cache

	// Attempt a rescan and ensure that no hashing occurs.
	hasher = &rescanHashProxy{hasher, t}
	snapshot, preservesExecutability, _, cache, _, err = Scan(
		root,
		nil,
		nil,
		hasher,
		cache,
		nil,
		nil,
		behavior.ProbeMode_ProbeModeProbe,
		SymlinkMode_SymlinkModePortable,
	)
	if !preservesExecutability {
		snapshot = PropagateExecutability(nil, testDirectory1Entry, snapshot)
	}
	if err != nil {
		t.Fatal("unable to rescan:", err)
	} else if cache == nil {
		t.Fatal("nil second cache returned")
	} else if !snapshot.Equal(testDirectory1Entry) {
		t.Error("second snapshot did not match expected")
	}

	// Verify that we haven't allocated any new cache entries on rescan.
	if len(cache.Entries) != len(oldCache.Entries) {
		t.Error("cache length mismatch for identical scans:", len(cache.Entries), "!=", len(oldCache.Entries))
	} else {
		for p, e := range cache.Entries {
			if oe, ok := oldCache.Entries[p]; !ok {
				t.Error("new cache content missing from old cache for path:", p)
			} else if e != oe {
				t.Error("new cache entry pointer does not match old cache entry pointer for path:", p)
			}
		}
	}
}

func TestScanCrossDeviceFail(t *testing.T) {
	// If we don't have the separate FAT32 partition mounted at a subdirectory,
	// skip this test.
	fat32Subroot := os.Getenv("MUTAGEN_TEST_FAT32_SUBROOT")
	if fat32Subroot == "" {
		t.Skip()
	}

	// Compute the subroot parent.
	parent := filepath.Dir(fat32Subroot)

	// Create a hasher.
	hasher := newTestHasher()

	// Perform a scan and ensure that it fails.
	if _, _, _, _, _, err := Scan(
		parent,
		nil,
		nil,
		hasher,
		nil,
		nil,
		nil,
		behavior.ProbeMode_ProbeModeProbe,
		SymlinkMode_SymlinkModePortable,
	); err == nil {
		t.Error("scan across device boundary did not fail")
	}
}
