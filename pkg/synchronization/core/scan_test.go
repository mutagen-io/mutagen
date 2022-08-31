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

// testingSnapshotStatistics computes the statistics fields that would be
// expected in a snapshot.
func testingSnapshotStatistics(entry *Entry, cache *Cache) (directoryCount, fileCount, symbolicLinkCount, totalFileSize uint64) {
	if entry != nil {
		entry.walk("", func(p string, e *Entry) {
			if e.Kind == EntryKind_Directory {
				directoryCount++
			} else if e.Kind == EntryKind_File {
				fileCount++
				totalFileSize += cache.Entries[p].Size
			} else if e.Kind == EntryKind_SymbolicLink {
				symbolicLinkCount++
			}
		}, false)
	}
	return
}

// TestScan tests Scan.
func TestScan(t *testing.T) {
	// Create contexts to use for tests.
	backgroundCtx := context.Background()
	cancelledCtx, cancel := context.WithCancel(backgroundCtx)
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
		// ctx is the context in which the operation should be performed.
		ctx context.Context
		// ignores are the ignore specifications to use for the test.
		ignores []string
		// symbolicLinkMode is the symbolic link mode to use for the test.
		symbolicLinkMode SymbolicLinkMode
		// permissionsMode is the permissions mode to use for the test.
		permissionsMode PermissionsMode
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
		{"root absent", nil, nil, nil, nil, nil, backgroundCtx, nil, SymbolicLinkMode_SymbolicLinkModePortable, PermissionsMode_PermissionsModePortable, false, nil, nil},
		{"root absent (manual permissions)", nil, nil, nil, nil, nil, backgroundCtx, nil, SymbolicLinkMode_SymbolicLinkModePortable, PermissionsMode_PermissionsModeManual, false, nil, nil},

		// Test a file root.
		{"file root", nil, tF1, tF1ContentMap, nil, nil, backgroundCtx, nil, SymbolicLinkMode_SymbolicLinkModePortable, PermissionsMode_PermissionsModePortable, false, tF1, nil},
		{"file root (manual permissions)", nil, tF1, tF1ContentMap, nil, nil, backgroundCtx, nil, SymbolicLinkMode_SymbolicLinkModePortable, PermissionsMode_PermissionsModeManual, false, tF1, nil},

		// Test an executable file root.
		{"executable file root", nil, tF3E, tF3ContentMap, nil, nil, backgroundCtx, nil, SymbolicLinkMode_SymbolicLinkModePortable, PermissionsMode_PermissionsModePortable, false, tF3E, nil},
		{"executable file root (manual permissions)", nil, tF3E, tF3ContentMap, nil, nil, backgroundCtx, nil, SymbolicLinkMode_SymbolicLinkModePortable, PermissionsMode_PermissionsModeManual, false, tF3, nil},

		// Test an empty directory root.
		{"empty directory root", nil, tD0, nil, nil, nil, backgroundCtx, nil, SymbolicLinkMode_SymbolicLinkModePortable, PermissionsMode_PermissionsModePortable, false, tD0, nil},
		{"empty directory root (manual permissions)", nil, tD0, nil, nil, nil, backgroundCtx, nil, SymbolicLinkMode_SymbolicLinkModePortable, PermissionsMode_PermissionsModeManual, false, tD0, nil},

		// Test a populated directory root.
		{"single file directory", nil, tD1, tD1ContentMap, nil, nil, backgroundCtx, nil, SymbolicLinkMode_SymbolicLinkModePortable, PermissionsMode_PermissionsModePortable, false, tD1, nil},
		{"single file directory (manual permissions)", nil, tD1, tD1ContentMap, nil, nil, backgroundCtx, nil, SymbolicLinkMode_SymbolicLinkModePortable, PermissionsMode_PermissionsModeManual, false, tD1, nil},

		// Test a directory with invalid filenames.
		{
			"directory with non-UTF-8 filename",
			func(f testingFilesystem) bool {
				// Most operating systems are too pedantic to allow this to
				// work, so we'll only run this on Linux with ext4.
				return !(runtime.GOOS == "linux" && f.name == "OS")
			},
			tD0, nil,
			func(root string) error {
				// We have to use tweaks to create the invalid filenames because
				// our transition infrastructure won't allow them.
				return os.WriteFile(
					// This is hellÖ encoded in ISO/IEC 8859-15 (the first four
					// characters of which are also valid ASCII/UTF-8).
					filepath.Join(root, string([]byte{0x68, 0x65, 0x6C, 0x6C, 0xD6})),
					[]byte(tF1Content),
					0600,
				)
			},
			nil,
			backgroundCtx,
			nil,
			SymbolicLinkMode_SymbolicLinkModePortable,
			PermissionsMode_PermissionsModePortable,
			false,
			&Entry{Contents: map[string]*Entry{
				"hell�": tPInvalidUTF8,
			}},
			nil,
		},

		// Test a populated directory root with executable contents.
		{"executable file directory", nil, tD3E, tD3ContentMap, nil, nil, backgroundCtx, nil, SymbolicLinkMode_SymbolicLinkModePortable, PermissionsMode_PermissionsModePortable, false, tD3E, nil},
		{"executable file directory (manual permissions)", nil, tD3E, tD3ContentMap, nil, nil, backgroundCtx, nil, SymbolicLinkMode_SymbolicLinkModePortable, PermissionsMode_PermissionsModeManual, false, tD3, nil},

		// Test a populated directory root with complex contents.
		{
			"complex directory (symbolic links ignored)",
			func(f testingFilesystem) bool {
				return f.name == "FAT32"
			},
			tDM, tDMContentMap,
			nil, nil,
			backgroundCtx,
			nil,
			SymbolicLinkMode_SymbolicLinkModeIgnore,
			PermissionsMode_PermissionsModePortable,
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
			backgroundCtx,
			nil,
			SymbolicLinkMode_SymbolicLinkModePortable,
			PermissionsMode_PermissionsModePortable,
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
			backgroundCtx,
			nil,
			SymbolicLinkMode_SymbolicLinkModePOSIXRaw,
			PermissionsMode_PermissionsModePortable,
			runtime.GOOS == "windows",
			tDM,
			nil,
		},

		// Test a directory root with ignored content.
		{"directory root with ignored content", nil, tD1, tD1ContentMap, nil, nil, backgroundCtx, []string{"file"}, SymbolicLinkMode_SymbolicLinkModePortable, PermissionsMode_PermissionsModePortable, false, nested("file", tU), nil},

		// Test with a cancelled context.
		// TODO: Figure out a way to cancel the context midway through reading
		// a file so that we can check for read preemption.
		{"cancelled context", nil, tD1, tD1ContentMap, nil, nil, cancelledCtx, nil, SymbolicLinkMode_SymbolicLinkModeIgnore, PermissionsMode_PermissionsModePortable, true, nil, nil},

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
			backgroundCtx,
			nil,
			SymbolicLinkMode_SymbolicLinkModePortable,
			PermissionsMode_PermissionsModePortable,
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
			backgroundCtx,
			nil,
			SymbolicLinkMode_SymbolicLinkModePortable,
			PermissionsMode_PermissionsModePortable,
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
			backgroundCtx,
			nil,
			SymbolicLinkMode_SymbolicLinkModePOSIXRaw,
			PermissionsMode_PermissionsModePortable,
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
			backgroundCtx,
			nil,
			SymbolicLinkMode_SymbolicLinkModePortable,
			PermissionsMode_PermissionsModePortable,
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
			backgroundCtx,
			nil,
			SymbolicLinkMode_SymbolicLinkModePortable,
			PermissionsMode_PermissionsModePortable,
			false,
			tD0,
			nil,
		},

		// Test invalid ignores.
		{"invalid ignore", nil, tD0, nil, nil, nil, backgroundCtx, []string{""}, SymbolicLinkMode_SymbolicLinkModePortable, PermissionsMode_PermissionsModePortable, true, nil, nil},

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
			backgroundCtx,
			nil,
			SymbolicLinkMode_SymbolicLinkModePortable,
			PermissionsMode_PermissionsModePortable,
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
			backgroundCtx,
			nil,
			SymbolicLinkMode_SymbolicLinkModePortable,
			PermissionsMode_PermissionsModePortable,
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
			backgroundCtx,
			nil,
			SymbolicLinkMode_SymbolicLinkModePortable,
			PermissionsMode_PermissionsModePortable,
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
			backgroundCtx,
			nil,
			SymbolicLinkMode_SymbolicLinkModePortable,
			PermissionsMode_PermissionsModePortable,
			false,
			nested("subdir", &Entry{Kind: EntryKind_Problematic, Problem: "*"}),
			nil,
		},

		// Test accelerated scanning with modifications, including cases where
		// problematic content is generated.
		{
			"file created", nil, tD1, tD1ContentMap, nil, nil, backgroundCtx, nil, SymbolicLinkMode_SymbolicLinkModePortable, PermissionsMode_PermissionsModePortable, false, tD1,
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
			"root type replaced", nil, tD0, nil, nil, nil, backgroundCtx, nil, SymbolicLinkMode_SymbolicLinkModePortable, PermissionsMode_PermissionsModePortable, false, tD0,
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
			backgroundCtx,
			nil,
			SymbolicLinkMode_SymbolicLinkModePortable,
			PermissionsMode_PermissionsModePortable,
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
			snapshot, cache, ignoreCache, err := Scan(
				test.ctx,
				root,
				nil, nil,
				hasher, nil,
				test.ignores, nil,
				behavior.ProbeMode_ProbeModeProbe,
				test.symbolicLinkMode,
				test.permissionsMode,
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
			if !snapshot.PreservesExecutability {
				snapshot.Content = PropagateExecutability(nil, test.expected, snapshot.Content)
			}

			// Compute expected statistics.
			directoryCount, fileCount, symbolicLinkCount, totalFileSize := testingSnapshotStatistics(
				snapshot.Content, cache,
			)

			// Check scan results.
			if !snapshot.Content.Equal(test.expected, true) {
				t.Errorf("%s: cold scan result not equal to expected on %s filesystem",
					test.description, filesystem.name,
				)
			}
			if cache == nil {
				t.Errorf("%s: nil cache returned by cold scan on %s filesystem",
					test.description, filesystem.name,
				)
			}
			if snapshot.Directories != directoryCount {
				t.Errorf("%s: cold scan directory count not equal to expected on %s filesystem: %d != %d",
					test.description, filesystem.name,
					snapshot.Directories, directoryCount,
				)
			}
			if snapshot.Files != fileCount {
				t.Errorf("%s: cold scan file count not equal to expected on %s filesystem: %d != %d",
					test.description, filesystem.name,
					snapshot.Files, fileCount,
				)
			}
			if snapshot.SymbolicLinks != symbolicLinkCount {
				t.Errorf("%s: cold scan symbolic link count not equal to expected on %s filesystem: %d != %d",
					test.description, filesystem.name,
					snapshot.SymbolicLinks, symbolicLinkCount,
				)
			}
			if snapshot.TotalFileSize != totalFileSize {
				t.Errorf("%s: cold scan total file size not equal to expected on %s filesystem: %d != %d",
					test.description, filesystem.name,
					snapshot.TotalFileSize, totalFileSize,
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
			newSnapshot, newCache, newIgnoreCache, err := Scan(
				test.ctx,
				root,
				nil, nil,
				rescanHasher, cache,
				test.ignores, ignoreCache,
				behavior.ProbeMode_ProbeModeProbe,
				test.symbolicLinkMode,
				test.permissionsMode,
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
			if !newSnapshot.PreservesExecutability {
				newSnapshot.Content = PropagateExecutability(nil, test.expected, newSnapshot.Content)
			}

			// Check scan results.
			if !newSnapshot.Equal(snapshot) {
				t.Errorf("%s: warm scan result not equal to cold scan on %s filesystem",
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
			if newSnapshot.Directories != directoryCount {
				t.Errorf("%s: warm scan directory count not equal to expected on %s filesystem: %d != %d",
					test.description, filesystem.name,
					newSnapshot.Directories, directoryCount,
				)
			}
			if newSnapshot.Files != fileCount {
				t.Errorf("%s: warm scan file count not equal to expected on %s filesystem: %d != %d",
					test.description, filesystem.name,
					newSnapshot.Files, fileCount,
				)
			}
			if newSnapshot.SymbolicLinks != symbolicLinkCount {
				t.Errorf("%s: warm scan symbolic link count not equal to expected on %s filesystem: %d != %d",
					test.description, filesystem.name,
					newSnapshot.SymbolicLinks, symbolicLinkCount,
				)
			}
			if newSnapshot.TotalFileSize != totalFileSize {
				t.Errorf("%s: warm scan total file size not equal to expected on %s filesystem: %d != %d",
					test.description, filesystem.name,
					newSnapshot.TotalFileSize, totalFileSize,
				)
			}

			// Perform an accelerated scan (without any re-check paths) using
			// the snapshot as a baseline.
			newSnapshot, newCache, newIgnoreCache, err = Scan(
				test.ctx,
				root,
				snapshot, nil,
				hasher, cache,
				test.ignores, ignoreCache,
				behavior.ProbeMode_ProbeModeProbe,
				test.symbolicLinkMode,
				test.permissionsMode,
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
			if !newSnapshot.PreservesExecutability {
				newSnapshot.Content = PropagateExecutability(nil, test.expected, newSnapshot.Content)
			}

			// Check scan results.
			if !newSnapshot.Equal(snapshot) {
				t.Errorf("%s: accelerated scan (without re-check paths) result not equal to cold scan on %s filesystem",
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
			if newSnapshot.Directories != directoryCount {
				t.Errorf("%s: accelerated scan (without re-check paths) directory count not equal to expected on %s filesystem: %d != %d",
					test.description, filesystem.name,
					newSnapshot.Directories, directoryCount,
				)
			}
			if newSnapshot.Files != fileCount {
				t.Errorf("%s: accelerated scan (without re-check paths) file count not equal to expected on %s filesystem: %d != %d",
					test.description, filesystem.name,
					newSnapshot.Files, fileCount,
				)
			}
			if newSnapshot.SymbolicLinks != symbolicLinkCount {
				t.Errorf("%s: accelerated scan (without re-check paths) symbolic link count not equal to expected on %s filesystem: %d != %d",
					test.description, filesystem.name,
					newSnapshot.SymbolicLinks, symbolicLinkCount,
				)
			}
			if newSnapshot.TotalFileSize != totalFileSize {
				t.Errorf("%s: accelerated scan (without re-check paths) total file size not equal to expected on %s filesystem: %d != %d",
					test.description, filesystem.name,
					newSnapshot.TotalFileSize, totalFileSize,
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
			newSnapshot, newCache, newIgnoreCache, err = Scan(
				test.ctx,
				root,
				snapshot, recheckPaths,
				hasher, cache,
				test.ignores, ignoreCache,
				behavior.ProbeMode_ProbeModeProbe,
				test.symbolicLinkMode,
				test.permissionsMode,
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
			if !newSnapshot.PreservesExecutability {
				newSnapshot.Content = PropagateExecutability(nil, modifiedExpected, newSnapshot.Content)
			}

			// Compute expected statistics.
			directoryCount, fileCount, symbolicLinkCount, totalFileSize = testingSnapshotStatistics(
				newSnapshot.Content, newCache,
			)

			// Check scan results. Since modifiers can perform arbitrary changes
			// to the root, we have to restrict certain checks to cases where
			// the filesystem hasn't been modified. We also have to decompose
			// our scan equivalence check since we're not doing a direct
			// comparison with the unmodified cold scan.
			if !newSnapshot.Content.Equal(modifiedExpected, true) {
				t.Errorf("%s: accelerated scan (with re-check path(s)) result not equal to expected on %s filesystem",
					test.description, filesystem.name,
				)
			}
			if newSnapshot.PreservesExecutability != snapshot.PreservesExecutability {
				t.Errorf("%s: accelerated scan (with re-check path(s)) differed in executability preservation behavior from cold scan on %s filesystem",
					test.description, filesystem.name,
				)
			}
			if newSnapshot.DecomposesUnicode != snapshot.DecomposesUnicode {
				t.Errorf("%s: accelerated scan (with re-check path(s)) differed in Unicode decomposition behavior from cold scan on %s filesystem",
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
			if newSnapshot.Directories != directoryCount {
				t.Errorf("%s: accelerated scan (with re-check path(s)) directory count not equal to expected on %s filesystem: %d != %d",
					test.description, filesystem.name,
					newSnapshot.Directories, directoryCount,
				)
			}
			if newSnapshot.Files != fileCount {
				t.Errorf("%s: accelerated scan (with re-check path(s)) file count not equal to expected on %s filesystem: %d != %d",
					test.description, filesystem.name,
					newSnapshot.Files, fileCount,
				)
			}
			if newSnapshot.SymbolicLinks != symbolicLinkCount {
				t.Errorf("%s: accelerated scan (with re-check path(s)) symbolic link count not equal to expected on %s filesystem: %d != %d",
					test.description, filesystem.name,
					newSnapshot.SymbolicLinks, symbolicLinkCount,
				)
			}
			if newSnapshot.TotalFileSize != totalFileSize {
				t.Errorf("%s: accelerated scan (with re-check path(s)) total file size not equal to expected on %s filesystem: %d != %d",
					test.description, filesystem.name,
					newSnapshot.TotalFileSize, totalFileSize,
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
	snapshot, _, _, err := Scan(
		context.Background(),
		parent,
		nil, nil,
		newTestingHasher(), nil,
		[]string{"*", "!" + name}, nil,
		behavior.ProbeMode_ProbeModeProbe,
		SymbolicLinkMode_SymbolicLinkModePortable,
		PermissionsMode_PermissionsModePortable,
	)
	if err != nil {
		t.Fatalf("unable to perform scan: %v", err)
	} else if snapshot == nil {
		t.Fatalf("scan returned nil result")
	}

	// Ensure that the result matches what we expect. We don't know what else
	// might be in the parent that ends up untracked, so we just check the entry
	// that corresponds to the filesystem crossing point.
	expected := &Entry{Kind: EntryKind_Problematic, Problem: "scan crossed filesystem boundary"}
	if !snapshot.Content.Contents[name].Equal(expected, true) {
		t.Errorf("result does not match expected: %v != %v", snapshot.Content.Contents[name], expected)
	}
}
