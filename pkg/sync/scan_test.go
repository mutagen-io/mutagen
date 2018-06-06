package sync

import (
	"crypto/sha1"
	"hash"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/filesystem"
)

func testCreateScanCycle(value ContentTestValue, ignores []string, symlinkMode SymlinkMode, expectEqual bool) error {
	// Create the content on disk and defer its removal.
	root, parent, err := value.CreateOnDisk()
	if err != nil {
		return errors.Wrap(err, "unable to create test content on disk")
	}
	defer os.RemoveAll(parent)

	// Grab the expected entry. If we're on a system that doesn't support
	// executability, then strip executability from the expected value.
	expected := value.entry()
	if !filesystem.PreservesExecutability {
		expected = StripExecutability(expected)
	}

	// Create a hasher.
	hasher := value.Hasher()

	// Perform a scan.
	snapshot, cache, err := Scan(root, hasher, nil, ignores, symlinkMode)
	if err != nil {
		return errors.Wrap(err, "unable to perform scan")
	} else if cache == nil {
		return errors.New("nil cache returned")
	} else if expectEqual && !snapshot.Equal(expected) {
		return errors.New("snapshot not equal to expected")
	} else if !expectEqual && snapshot.Equal(expected) {
		return errors.New("snapshot should not have equaled original")
	}

	// Success.
	return nil
}

func TestScanNilRoot(t *testing.T) {
	if err := testCreateScanCycle(ContentTestValueNil, nil, SymlinkMode_Sane, true); err != nil {
		t.Error("creation/scan cycle failed:", err)
	}
}

func TestScanFile1Root(t *testing.T) {
	if err := testCreateScanCycle(ContentTestValueFile1, nil, SymlinkMode_Sane, true); err != nil {
		t.Error("creation/scan cycle failed:", err)
	}
}

func TestScanFile2Root(t *testing.T) {
	if err := testCreateScanCycle(ContentTestValueFile2, nil, SymlinkMode_Sane, true); err != nil {
		t.Error("creation/scan cycle failed:", err)
	}
}

func TestScanFile3Root(t *testing.T) {
	if err := testCreateScanCycle(ContentTestValueFile3, nil, SymlinkMode_Sane, true); err != nil {
		t.Error("creation/scan cycle failed:", err)
	}
}

func TestScanDirectory1Root(t *testing.T) {
	if err := testCreateScanCycle(ContentTestValueDirectory1, nil, SymlinkMode_Sane, true); err != nil {
		t.Error("creation/scan cycle failed:", err)
	}
}

func TestScanDirectory2Root(t *testing.T) {
	if err := testCreateScanCycle(ContentTestValueDirectory2, nil, SymlinkMode_Sane, true); err != nil {
		t.Error("creation/scan cycle failed:", err)
	}
}

func TestScanDirectory3Root(t *testing.T) {
	if err := testCreateScanCycle(ContentTestValueDirectory3, nil, SymlinkMode_Sane, true); err != nil {
		t.Error("creation/scan cycle failed:", err)
	}
}

func TestScanDirectorySaneSymlinkSane(t *testing.T) {
	if err := testCreateScanCycle(ContentTestValueDirectoryWithSaneSymlink, nil, SymlinkMode_Sane, true); err != nil {
		t.Error("sane symlink not allowed inside root with sane symlink mode:", err)
	}
}

func TestScanDirectorySaneSymlinkIgnore(t *testing.T) {
	if err := testCreateScanCycle(ContentTestValueDirectoryWithSaneSymlink, nil, SymlinkMode_Ignore, false); err != nil {
		t.Error("sane symlink not allowed inside root with ignore symlink mode:", err)
	}
}

func TestScanDirectorySaneSymlinkPOSIXRaw(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	if err := testCreateScanCycle(ContentTestValueDirectoryWithSaneSymlink, nil, SymlinkMode_POSIXRaw, true); err != nil {
		t.Error("sane symlink not allowed inside root with POSIX raw symlink mode:", err)
	}
}

func TestScanDirectoryInvalidSymlinkNotSane(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	if testCreateScanCycle(ContentTestValueDirectoryWithInvalidSymlink, nil, SymlinkMode_Sane, true) == nil {
		t.Error("invalid symlink allowed inside root with sane symlink mode")
	}
}

func TestScanDirectoryInvalidSymlinkIgnore(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	if err := testCreateScanCycle(ContentTestValueDirectoryWithInvalidSymlink, nil, SymlinkMode_Ignore, false); err != nil {
		t.Error("invalid symlink not allowed inside root with ignore symlink mode:", err)
	}
}

func TestScanDirectoryInvalidSymlinkPOSIXRaw(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	if err := testCreateScanCycle(ContentTestValueDirectoryWithInvalidSymlink, nil, SymlinkMode_POSIXRaw, true); err != nil {
		t.Error("invalid symlink not allowed inside root with POSIX raw symlink mode:", err)
	}
}

func TestScanDirectoryEscapingSymlinkSane(t *testing.T) {
	if testCreateScanCycle(ContentTestValueDirectoryWithEscapingSymlink, nil, SymlinkMode_Sane, true) == nil {
		t.Error("escaping symlink allowed inside root with sane symlink mode")
	}
}

func TestScanDirectoryEscapingSymlinkIgnore(t *testing.T) {
	if err := testCreateScanCycle(ContentTestValueDirectoryWithEscapingSymlink, nil, SymlinkMode_Ignore, false); err != nil {
		t.Error("escaping symlink not allowed inside root with ignore symlink mode:", err)
	}
}

func TestScanDirectoryEscapingSymlinkPOSIXRaw(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	if err := testCreateScanCycle(ContentTestValueDirectoryWithEscapingSymlink, nil, SymlinkMode_POSIXRaw, true); err != nil {
		t.Error("escaping symlink not allowed inside root with POSIX raw symlink mode:", err)
	}
}

func TestScanDirectoryAbsoluteSymlinkSane(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	if testCreateScanCycle(ContentTestValueDirectoryWithAbsoluteSymlink, nil, SymlinkMode_Sane, true) == nil {
		t.Error("escaping symlink allowed inside root with sane symlink mode")
	}
}

func TestScanDirectoryAbsoluteSymlinkIgnore(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	if err := testCreateScanCycle(ContentTestValueDirectoryWithAbsoluteSymlink, nil, SymlinkMode_Ignore, false); err != nil {
		t.Error("escaping symlink not allowed inside root with ignore symlink mode:", err)
	}
}

func TestScanDirectoryAbsoluteSymlinkPOSIXRaw(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	if err := testCreateScanCycle(ContentTestValueDirectoryWithAbsoluteSymlink, nil, SymlinkMode_POSIXRaw, true); err != nil {
		t.Error("escaping symlink not allowed inside root with POSIX raw symlink mode:", err)
	}
}

func TestScanPOSIXRawNotAllowedOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip()
	}
	if testCreateScanCycle(ContentTestValueDirectoryWithSaneSymlink, nil, SymlinkMode_POSIXRaw, true) == nil {
		t.Error("POSIX raw symlink mode allowed for scan on Windows")
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
	if _, _, err := Scan(root, sha1.New(), nil, nil, SymlinkMode_Sane); err == nil {
		t.Error("scan of symlink root allowed")
	}
}

func TestScanInvalidIgnores(t *testing.T) {
	if testCreateScanCycle(ContentTestValueDirectory1, []string{""}, SymlinkMode_Sane, true) == nil {
		t.Error("scan allowed with invalid ignore specification")
	}
}

func TestScanIgnore(t *testing.T) {
	if err := testCreateScanCycle(ContentTestValueDirectory1, []string{"second directory"}, SymlinkMode_Sane, false); err != nil {
		t.Error("unexpected result when ignoring directory:", err)
	}
}

func TestScanIgnoreDirectory(t *testing.T) {
	if err := testCreateScanCycle(ContentTestValueDirectory1, []string{"directory/"}, SymlinkMode_Sane, false); err != nil {
		t.Error("unexpected result when ignoring directory:", err)
	}
}

func TestScanFileNotIgnoredOnDirectorySpecification(t *testing.T) {
	if err := testCreateScanCycle(ContentTestValueDirectory1, []string{"file/"}, SymlinkMode_Sane, true); err != nil {
		t.Error("unexpected result when ignoring directory:", err)
	}
}

func TestScanSubfileNotIgnoredOnRootSpecification(t *testing.T) {
	if err := testCreateScanCycle(ContentTestValueDirectory1, []string{"/subfile.exe"}, SymlinkMode_Sane, true); err != nil {
		t.Error("unexpected result when ignoring directory:", err)
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
	// Create the content on disk and defer its removal.
	root, parent, err := ContentTestValueDirectory1.CreateOnDisk()
	if err != nil {
		t.Fatal("unable to create test content on disk:", err)
	}
	defer os.RemoveAll(parent)

	// Grab the expected entry. If we're on a system that doesn't support
	// executability, then strip executability from the expected value.
	expected := ContentTestValueDirectory1.entry()
	if !filesystem.PreservesExecutability {
		expected = StripExecutability(expected)
	}

	// Create a hasher.
	hasher := ContentTestValueDirectory1.Hasher()

	// Create an initial snapshot and validate the results.
	snapshot, cache, err := Scan(root, hasher, nil, nil, SymlinkMode_Sane)
	if err != nil {
		t.Fatal("unable to create snapshot:", err)
	} else if cache == nil {
		t.Fatal("nil cache returned")
	} else if !snapshot.Equal(expected) {
		t.Error("snapshot did not match expected")
	}

	// Attempt a rescan and ensure that no hashing occurs.
	hasher = &rescanHashProxy{hasher, t}
	if snapshot, cache, err = Scan(root, hasher, cache, nil, SymlinkMode_Sane); err != nil {
		t.Fatal("unable to rescan:", err)
	} else if cache == nil {
		t.Fatal("nil second cache returned")
	} else if !snapshot.Equal(expected) {
		t.Error("second snapshot did not match expected")
	}
}
