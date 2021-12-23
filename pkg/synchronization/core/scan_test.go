package core

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/filesystem/behavior"
)

// testingIgnoreCachesEqual verifies that two ignore caches are equal.
func testingIgnoreCachesEqual(first, second IgnoreCache) bool {
	// Check lengths.
	if len(first) != len(second) {
		return false
	}

	// Check contents.
	for key, f := range first {
		if s, ok := second[key]; !ok || s != f {
			return false
		}
	}

	// Done.
	return true
}

// testingAcceleratedIgnoreCacheIsSubset verifies that an accelerated ignore
// cache is a subset of original, excluding the presence of a root path key in
// the accelerated case.
func testingAcceleratedIgnoreCacheIsSubset(accelerated, original IgnoreCache) bool {
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

// TestScan tests Scan.
func TestScan(t *testing.T) {
	// Create contexts to use for tests.
	background := context.Background()
	cancelled, cancel := context.WithCancel(background)
	cancel()

	// Check if the platform supports empty symbolic link targets. Unfortunately
	// this is wildly inconsistent. macOS, for example, supports them in 11.0
	// but not earlier. Linux doesn't seem to have ever supported them and gives
	// a strange error code when trying to create them. More information can be
	// found here: https://lwn.net/Articles/551224/. The best option is just to
	// test for support. Manpages don't seem to accurately describe support.
	emptySymbolicLinksSupported := os.Symlink("", filepath.Join(t.TempDir(), "l")) == nil

	// Define test cases.
	tests := []struct {
		// description is a human readable description of the test case.
		description string
		// skip is a callback that can be used to skip a test on a particular
		// filesystem (or based on other runtime criteria).
		skip func(testingFilesystem) bool
		// baseline is the baseline content to create on disk.
		baseline *Entry
		// baselineContentMap is the content map for baseline.
		baselineContentMap testingContentMap
		// tweak is an optional callback that can be used to perform additional
		// tweaks to generated on-disk content. For more information about the
		// role of tweak, see the corresponding member in testingContentManager.
		tweak func(string) error
		// untweak is an optional callback that can be used to fix content for
		// successful removal by os.RemoveAll. For more information about the
		// role of untweak, see the untweak member in testingContentManager. In
		// this case, the untweak function can (and should) also be used to fix
		// changes made by modifier that would prevent content removal.
		untweak func(string) error
		// context is the context in which the operation should be performed.
		context context.Context
		// ignores are the ignore specifications to use for the test.
		ignores []string
		// symbolicLinkMode is the symbolic link mode to use for the test.
		symbolicLinkMode SymbolicLinkMode
		// expectFailure indicates whether or not scan failure is expected.
		expectFailure bool
		// expected is the expected result of the scan.
		expected *Entry
		// modifier is an optional callback that allows tests to perform on-disk
		// file modifications between scans to use as a test of accelerated
		// scanning. The function should return a list of changes that will be
		// applied to expected in order to compute the new expected results. The
		// Path member of each change will be used as a recheck path for
		// accelerated scanning, and thus changes should be in decomposed form.
		modifier func(string) ([]*Change, error)
	}{
		// Test an absence of content.
		{"root absent", nil, nil, nil, nil, nil, background, nil, SymbolicLinkMode_SymbolicLinkModePortable, false, nil, nil},

		// Test a file root.
		{"file root", nil, tF1, tF1ContentMap, nil, nil, background, nil, SymbolicLinkMode_SymbolicLinkModePortable, false, tF1, nil},

		// Test an empty directory root.
		{"empty directory root", nil, tD0, nil, nil, nil, background, nil, SymbolicLinkMode_SymbolicLinkModePortable, false, tD0, nil},

		// Test a populated directory root.
		{"single file directory", nil, tD1, tD1ContentMap, nil, nil, background, nil, SymbolicLinkMode_SymbolicLinkModePortable, false, tD1, nil},

		// Test a populated directory root with complex contents.
		{
			"complex directory (symbolic links ignored)",
			func(f testingFilesystem) bool {
				return f.name == "FAT32"
			},
			tDM, tDMContentMap,
			nil, nil,
			background,
			nil,
			SymbolicLinkMode_SymbolicLinkModeIgnore,
			false,
			tDMSU,
			nil,
		},
		{
			"complex directory (portable symbolic links)",
			func(f testingFilesystem) bool {
				return f.name == "FAT32"
			},
			tDM, tDMContentMap,
			nil, nil,
			background,
			nil,
			SymbolicLinkMode_SymbolicLinkModePortable,
			false,
			tDM,
			nil,
		},
		{
			"complex directory (POSIX raw symbolic links)",
			func(f testingFilesystem) bool {
				return f.name == "FAT32"
			},
			tDM, tDMContentMap,
			nil, nil,
			background,
			nil,
			SymbolicLinkMode_SymbolicLinkModePOSIXRaw,
			runtime.GOOS == "windows",
			tDM,
			nil,
		},

		// Test a directory root with ignored content.
		{"directory root with ignored content", nil, tD1, tD1ContentMap, nil, nil, background, []string{"file"}, SymbolicLinkMode_SymbolicLinkModePortable, false, nested("file", tU), nil},

		// Test with a cancelled context.
		// TODO: Figure out a way to cancel the context midway through reading
		// a file so that we can check for read preemption.
		{"cancelled context", nil, tD1, tD1ContentMap, nil, nil, cancelled, nil, SymbolicLinkMode_SymbolicLinkModeIgnore, true, nil, nil},

		// Test a directory with an untracked Unix domain socket.
		{
			"untracked Unix domain socket",
			func(f testingFilesystem) bool {
				// TODO: We should remove this Windows check and only skip this
				// test if running on a FAT32 filesystem. Unfortunately this is
				// blocked by golang/go#33357, but once that's resolved this
				// test should work automatically, assuming they also fix it for
				// os.File.Readdir as well (or that there's some way we can
				// adjust our Directory implementation to get at it).
				return runtime.GOOS == "windows" || f.name == "FAT32"
			},
			tD0, nil,
			func(root string) error {
				listener, err := net.Listen("unix", filepath.Join(root, "socket.sock"))
				if err != nil {
					return fmt.Errorf("unable to create socket: %w", err)
				}
				unixListener, ok := listener.(*net.UnixListener)
				if !ok {
					listener.Close()
					return errors.New("listener was not a Unix listener")
				}
				unixListener.SetUnlinkOnClose(false)
				unixListener.Close()
				return nil
			},
			nil,
			background,
			nil,
			SymbolicLinkMode_SymbolicLinkModePortable,
			false,
			nested("socket.sock", tU),
			nil,
		},

		// Test invalid symbolic link targets.
		{
			"invalid empty symbolic link (portable symbolic links)",
			func(f testingFilesystem) bool {
				return !emptySymbolicLinksSupported || f.name == "FAT32"
			},
			tD0, nil,
			func(root string) error {
				return os.Symlink("", filepath.Join(root, "badlink"))
			},
			nil,
			background,
			nil,
			SymbolicLinkMode_SymbolicLinkModePortable,
			false,
			nested("badlink", &Entry{Kind: EntryKind_Problematic, Problem: "*"}),
			nil,
		},
		{
			"invalid empty symbolic link (POSIX raw symbolic links)",
			func(f testingFilesystem) bool {
				return !emptySymbolicLinksSupported || f.name == "FAT32"
			},
			tD0, nil,
			func(root string) error {
				return os.Symlink("", filepath.Join(root, "badlink"))
			},
			nil,
			background,
			nil,
			SymbolicLinkMode_SymbolicLinkModePOSIXRaw,
			false,
			nested("badlink", &Entry{Kind: EntryKind_Problematic, Problem: "*"}),
			nil,
		},
		{
			"invalid absolute symbolic link (portable symbolic links)",
			func(f testingFilesystem) bool {
				return runtime.GOOS == "windows" || f.name != "OS"
			},
			tD0, nil,
			func(root string) error {
				return os.Symlink("/", filepath.Join(root, "badlink"))
			},
			nil,
			background,
			nil,
			SymbolicLinkMode_SymbolicLinkModePortable,
			false,
			nested("badlink", &Entry{Kind: EntryKind_Problematic, Problem: "*"}),
			nil,
		},

		// Test with Mutagen temporary files.
		{
			"temporary file ignore",
			nil,
			tD0, nil,
			func(root string) error {
				return os.WriteFile(filepath.Join(root, filesystem.TemporaryNamePrefix+"test"), nil, 0600)
			},
			nil,
			background,
			nil,
			SymbolicLinkMode_SymbolicLinkModePortable,
			false,
			tD0,
			nil,
		},

		// Test invalid ignores.
		{"invalid ignore", nil, tD0, nil, nil, nil, background, []string{""}, SymbolicLinkMode_SymbolicLinkModePortable, true, nil, nil},

		// Test unreadable content.
		{
			"unreadable file root",
			func(f testingFilesystem) bool {
				return runtime.GOOS == "windows" || f.name != "OS"
			},
			tF1, tF1ContentMap,
			func(root string) error {
				return os.Chmod(root, 0200)
			},
			func(root string) error {
				return os.Chmod(root, 0600)
			},
			background,
			nil,
			SymbolicLinkMode_SymbolicLinkModePortable,
			true,
			nil,
			nil,
		},
		{
			"unreadable directory root",
			func(f testingFilesystem) bool {
				return runtime.GOOS == "windows" || f.name != "OS"
			},
			tD0, nil,
			func(root string) error {
				return os.Chmod(root, 0300)
			},
			func(root string) error {
				return os.Chmod(root, 0700)
			},
			background,
			nil,
			SymbolicLinkMode_SymbolicLinkModePortable,
			true,
			nil,
			nil,
		},
		{
			"unreadable file content",
			func(f testingFilesystem) bool {
				return runtime.GOOS == "windows" || f.name != "OS"
			},
			tD1, tD1ContentMap,
			func(root string) error {
				return os.Chmod(filepath.Join(root, "file"), 0200)
			},
			func(root string) error {
				return os.Chmod(filepath.Join(root, "file"), 0600)
			},
			background,
			nil,
			SymbolicLinkMode_SymbolicLinkModePortable,
			false,
			nested("file", &Entry{Kind: EntryKind_Problematic, Problem: "*"}),
			nil,
		},
		{
			"unreadable directory content",
			func(f testingFilesystem) bool {
				return runtime.GOOS == "windows" || f.name != "OS"
			},
			tD0, nil,
			func(root string) error {
				return os.Mkdir(filepath.Join(root, "subdir"), 0300)
			},
			func(root string) error {
				return os.Chmod(filepath.Join(root, "subdir"), 0700)
			},
			background,
			nil,
			SymbolicLinkMode_SymbolicLinkModePortable,
			false,
			nested("subdir", &Entry{Kind: EntryKind_Problematic, Problem: "*"}),
			nil,
		},

		// Test accelerated scanning with modifications, including cases where
		// problematic content is generated.
		{
			"file created", nil, tD1, tD1ContentMap, nil, nil, background, nil, SymbolicLinkMode_SymbolicLinkModePortable, false, tD1,
			func(root string) ([]*Change, error) {
				if err := os.WriteFile(filepath.Join(root, "newfile"), []byte(tF2Content), 0600); err != nil {
					return nil, err
				}
				return []*Change{
					{Path: "newfile", New: tF2},
				}, nil
			},
		},
		{
			"root type replaced", nil, tD0, nil, nil, nil, background, nil, SymbolicLinkMode_SymbolicLinkModePortable, false, tD0,
			func(root string) ([]*Change, error) {
				if err := os.Remove(root); err != nil {
					return nil, err
				} else if err := os.WriteFile(root, []byte(tF2Content), 0600); err != nil {
					return nil, err
				}
				return []*Change{
					{Old: tD0, New: tF2},
				}, nil
			},
		},
		{
			"file made unreadable",
			func(f testingFilesystem) bool {
				return runtime.GOOS == "windows" || f.name != "OS"
			},
			tD1, tD1ContentMap,
			nil,
			func(root string) error {
				return os.Chmod(filepath.Join(root, "file"), 0600)
			},
			background,
			nil,
			SymbolicLinkMode_SymbolicLinkModePortable,
			false,
			tD1,
			func(root string) ([]*Change, error) {
				soon := time.Now().Add(10 * time.Second)
				if err := os.Chtimes(filepath.Join(root, "file"), soon, soon); err != nil {
					return nil, err
				}
				if err := os.Chmod(filepath.Join(root, "file"), 0200); err != nil {
					return nil, err
				}
				return []*Change{{
					Path: "file",
					Old:  tF1,
					New:  &Entry{Kind: EntryKind_Problematic, Problem: "*"},
				}}, nil
			},
		},
	}

	// Create a hasher that we can use for testing.
	hasher := newTestingHasher()

	// Process test cases for every filesystem.
	for _, filesystem := range testingFilesystems {
		for _, test := range tests {
			// Check if this test is skipped on this platform or filesystem.
			if test.skip != nil && test.skip(filesystem) {
				continue
			}

			// Generate content for this test.
			generator := &testingContentManager{
				storage:            filesystem.storage,
				baseline:           test.baseline,
				baselineContentMap: test.baselineContentMap,
				tweak:              test.tweak,
				untweak:            test.untweak,
			}
			root, err := generator.generate()
			if err != nil {
				t.Errorf("%s: unable to generate test content on %s filesystem: %v",
					test.description, filesystem.name, err,
				)
				continue
			}

			// Define a cleanup function that will report errors.
			cleanup := func() {
				if err := generator.remove(); err != nil {
					t.Errorf("%s: unable to remove test content on %s filesystem: %v",
						test.description, filesystem.name, err,
					)
				}
			}

			// Perform a cold scan and handle failure cases.
			snapshot, preservesExecutability, decomposesUnicode, cache, ignoreCache, err := Scan(
				test.context,
				root,
				nil, nil,
				hasher, nil,
				test.ignores, IgnorerMode_IgnorerModeDefault, nil,
				behavior.ProbeMode_ProbeModeProbe,
				test.symbolicLinkMode,
			)
			if test.expectFailure {
				if err == nil {
					t.Errorf("%s: cold scan succeeded unexpectedly on %s filesystem",
						test.description, filesystem.name,
					)
				}
				cleanup()
				continue
			} else if err != nil {
				t.Errorf("%s: cold scan failed on %s filesystem: %v",
					test.description, filesystem.name, err,
				)
				cleanup()
				continue
			}

			// Propagate executability if the filesystem doesn't preserve it.
			if !preservesExecutability {
				snapshot = PropagateExecutability(nil, test.expected, snapshot)
			}

			// Check scan results.
			if !snapshot.Equal(test.expected, true) {
				t.Errorf("%s: cold scan result not equal to expected on %s filesystem",
					test.description, filesystem.name,
				)
			}
			if cache == nil {
				t.Errorf("%s: nil cache returned by cold scan on %s filesystem",
					test.description, filesystem.name,
				)
			}

			// Create a proxy hasher to track re-hashing.
			rescanHasher := &testHashingDetector{
				hasher, func() {
					t.Errorf("%s: hashing occurred on warm scan on %s filesystem",
						test.description, filesystem.name,
					)
				},
			}

			// Perform a warm (but non-accelerated) scan.
			newSnapshot, newPreservesExecutability, newDecomposesUnicode, newCache, newIgnoreCache, err := Scan(
				test.context,
				root,
				nil, nil,
				rescanHasher, cache,
				test.ignores, IgnorerMode_IgnorerModeDefault, ignoreCache,
				behavior.ProbeMode_ProbeModeProbe,
				test.symbolicLinkMode,
			)

			// Handle scan failure (which isn't expected at this point).
			if err != nil {
				t.Errorf("%s: warm scan failed on %s filesystem: %v",
					test.description, filesystem.name, err,
				)
				cleanup()
				continue
			}

			// Propagate executability if the filesystem doesn't preserve it.
			if !newPreservesExecutability {
				newSnapshot = PropagateExecutability(nil, test.expected, newSnapshot)
			}

			// Check scan results.
			if !newSnapshot.Equal(test.expected, true) {
				t.Errorf("%s: warm scan result not equal to expected on %s filesystem",
					test.description, filesystem.name,
				)
			}
			if newCache == nil {
				t.Errorf("%s: nil cache returned by warm scan on %s filesystem",
					test.description, filesystem.name,
				)
			} else if !newCache.Equal(cache) {
				t.Errorf("%s: warm scan cache does not match baseline cache on %s filesystem",
					test.description, filesystem.name,
				)
			}
			if !testingIgnoreCachesEqual(newIgnoreCache, ignoreCache) {
				t.Errorf("%s: warm scan ignore cache does not match baseline on %s filesystem",
					test.description, filesystem.name,
				)
			}
			if newPreservesExecutability != preservesExecutability {
				t.Errorf("%s: warm scan differed in executability preservation behavior on %s filesystem",
					test.description, filesystem.name,
				)
			}
			if newDecomposesUnicode != decomposesUnicode {
				t.Errorf("%s: warm scan differed in Unicode decomposition behavior on %s filesystem",
					test.description, filesystem.name,
				)
			}

			// Perform an accelerated scan (without any re-check paths) using
			// the snapshot as a baseline.
			newSnapshot, newPreservesExecutability, newDecomposesUnicode, newCache, newIgnoreCache, err = Scan(
				test.context,
				root,
				snapshot, nil,
				hasher, cache,
				test.ignores, IgnorerMode_IgnorerModeDefault, ignoreCache,
				behavior.ProbeMode_ProbeModeProbe,
				test.symbolicLinkMode,
			)

			// Handle scan failure (which isn't expected at this point).
			if err != nil {
				t.Errorf("%s: accelerated scan (without re-check paths) failed on %s filesystem: %v",
					test.description, filesystem.name, err,
				)
				cleanup()
				continue
			}

			// Propagate executability if the filesystem doesn't preserve it.
			if !newPreservesExecutability {
				newSnapshot = PropagateExecutability(nil, test.expected, newSnapshot)
			}

			// Check scan results.
			if !newSnapshot.Equal(test.expected, true) {
				t.Errorf("%s: accelerated scan (without re-check paths) result not equal to expected on %s filesystem",
					test.description, filesystem.name,
				)
			}
			if newCache == nil {
				t.Errorf("%s: nil cache returned by accelerated scan (without re-check paths) on %s filesystem",
					test.description, filesystem.name,
				)
			} else if !newCache.Equal(cache) {
				t.Errorf("%s: accelerated scan (without re-check paths) cache does not match baseline cache on %s filesystem",
					test.description, filesystem.name,
				)
			}
			if !testingAcceleratedIgnoreCacheIsSubset(newIgnoreCache, ignoreCache) {
				t.Errorf("%s: accelerated scan (without re-check paths) ignore cache not a subset of baseline on %s filesystem",
					test.description, filesystem.name,
				)
			}
			if newPreservesExecutability != preservesExecutability {
				t.Errorf("%s: accelerated scan (without re-check paths) differed in executability preservation behavior on %s filesystem",
					test.description, filesystem.name,
				)
			}
			if newDecomposesUnicode != decomposesUnicode {
				t.Errorf("%s: accelerated scan (without re-check paths) differed in Unicode decomposition behavior on %s filesystem",
					test.description, filesystem.name,
				)
			}

			// If a modifier has been specified, then use it to modify the root
			// and generate changes to the expected entry. We'll also use those
			// change paths to populate the recheck path map. If no modifier has
			// been specified, then just re-use the expected entry but still run
			// the scan with a (bogus) recheck path.
			var recheckPaths map[string]bool
			var modifiedExpected *Entry
			if test.modifier != nil {
				if changes, err := test.modifier(root); err != nil {
					t.Errorf("%s: unable to perform modifications on %s filesystem: %v", test.description, filesystem.name, err)
					cleanup()
					continue
				} else if modifiedExpected, err = Apply(test.expected, changes); err != nil {
					t.Errorf("%s: unable to apply expected entry changes on %s filesystem: %v", test.description, filesystem.name, err)
					cleanup()
					continue
				} else {
					recheckPaths = make(map[string]bool, len(changes))
					for _, change := range changes {
						recheckPaths[change.Path] = true
					}
				}
			} else {
				recheckPaths = map[string]bool{"non/existent/testing/path": true}
				modifiedExpected = test.expected
			}

			// Perform an accelerated scan (with re-check paths) using the
			// snapshot as a baseline.
			newSnapshot, newPreservesExecutability, newDecomposesUnicode, newCache, newIgnoreCache, err = Scan(
				test.context,
				root,
				snapshot, recheckPaths,
				hasher, cache,
				test.ignores, IgnorerMode_IgnorerModeDefault, ignoreCache,
				behavior.ProbeMode_ProbeModeProbe,
				test.symbolicLinkMode,
			)

			// Handle scan failure (which isn't expected at this point).
			if err != nil {
				t.Errorf("%s: accelerated scan (with re-check path) failed on %s filesystem: %v",
					test.description, filesystem.name, err,
				)
				cleanup()
				continue
			}

			// Propagate executability if the filesystem doesn't preserve it.
			if !newPreservesExecutability {
				newSnapshot = PropagateExecutability(nil, modifiedExpected, newSnapshot)
			}

			// Check scan results. Since modifiers can perform arbitrary changes
			// to the root, we have to restrict certain checks to cases where
			// the filesystem hasn't been modified. In the case of Unicode
			// decomposition checks, we have to restrict to the unmodified case
			// because a modifier might change the root from (say) a directory
			// to a file, in which case scan won't probe for Unicode behavior.
			if !newSnapshot.Equal(modifiedExpected, true) {
				t.Errorf("%s: accelerated scan (with re-check path(s)) result not equal to expected on %s filesystem",
					test.description, filesystem.name,
				)
			}
			if newCache == nil {
				t.Errorf("%s: nil cache returned by accelerated scan (with re-check path(s)) on %s filesystem",
					test.description, filesystem.name,
				)
			} else if test.modifier == nil && !newCache.Equal(cache) {
				t.Errorf("%s: accelerated scan (with re-check path(s)) cache does not match baseline cache on %s filesystem",
					test.description, filesystem.name,
				)
			}
			if test.modifier == nil && !testingAcceleratedIgnoreCacheIsSubset(newIgnoreCache, ignoreCache) {
				t.Errorf("%s: accelerated scan (with re-check path(s)) ignore cache not a subset of baseline on %s filesystem",
					test.description, filesystem.name,
				)
			}
			if newPreservesExecutability != preservesExecutability {
				t.Errorf("%s: accelerated scan (with re-check path(s)) differed in executability preservation behavior on %s filesystem",
					test.description, filesystem.name,
				)
			}
			if test.modifier == nil && newDecomposesUnicode != decomposesUnicode {
				t.Errorf("%s: accelerated scan (with re-check path(s)) differed in Unicode decomposition behavior on %s filesystem",
					test.description, filesystem.name,
				)
			}

			// Perform cleanup.
			cleanup()
		}
	}
}

// TestScanCrossFilesystemBoundary tests the behavior of Scan when crossing a
// filesystem boundary. This test uses the APFS test partition on Darwin, if
// available. It is separate from TestScan simply because it's tedious to
// generate (and mount/unmount) disk images in Go, meaning that this test is
// hard to do as a tweak/untweak operation in a standard scan test.
func TestScanCrossFilesystemBoundary(t *testing.T) {
	// If we don't have a filesystem mounted within another filesystem, then
	// skip this test.
	crossing := os.Getenv("MUTAGEN_TEST_SUBFS_ROOT")
	if crossing == "" {
		t.Skip()
	}

	// Compute the parent path and the name of the crossing point.
	parent, name := filepath.Split(crossing)

	// Perform a scan that crosses the boundary. We'll ignore everything else in
	// the parent other than the crossing point.
	result, _, _, _, _, err := Scan(
		context.Background(),
		parent,
		nil, nil,
		newTestingHasher(),
		nil,
		[]string{"*", "!" + name},
		IgnorerMode_IgnorerModeDefault,
		nil,
		behavior.ProbeMode_ProbeModeProbe,
		SymbolicLinkMode_SymbolicLinkModePortable,
	)
	if err != nil {
		t.Fatalf("unable to perform scan: %v", err)
	} else if result == nil {
		t.Fatalf("scan returned nil result")
	}

	// Ensure that the result matches what we expect. We don't know what else
	// might be in the parent that ends up untracked, so we just check the entry
	// that corresponds to the filesystem crossing point.
	expected := &Entry{Kind: EntryKind_Problematic, Problem: "scan crossed filesystem boundary"}
	if !result.Contents[name].Equal(expected, true) {
		t.Errorf("result does not match expected: %v != %v", result.Contents[name], expected)
	}
}
