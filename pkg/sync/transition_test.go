package sync

import (
	"bytes"
	"hash"
	"io/ioutil"
	"os"
	pathpkg "path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/filesystem"
)

// testEntryDecomposer provides the implementation for testDecomposeEntry.
type testEntryDecomposer struct {
	// creation records whether or not the decomposition is for a creation
	// transition (as opposed to a removal transition).
	creation bool
	// transitions is the accumulated list of decomposed transitions.
	transitions []*Change
}

// decompose decomposes an entry recursively and records associated transitions.
func (d *testEntryDecomposer) decompose(path string, entry *Entry) {
	// If the entry is non-existent, then there are no transitions.
	if entry == nil {
		return
	}

	// Create a shallow copy of the entry.
	shallowEntry := entry.CopyShallow()

	// If this is a creation decomposition, add this entry before processing any
	// contents.
	if d.creation {
		d.transitions = append(d.transitions, &Change{Path: path, New: shallowEntry})
	}

	// If this is a directory, handle its contents.
	if entry.Kind == EntryKind_Directory {
		for name, entry := range entry.Contents {
			d.decompose(pathpkg.Join(path, name), entry)
		}
	}

	// If this is a removal decomposition, add this entry after processing any
	// contents.
	if !d.creation {
		d.transitions = append(d.transitions, &Change{Path: path, Old: shallowEntry})
	}
}

// testDecomposeEntry decomposes an entry into a sequence of transitions that
// can more granularly test Transition. Instead of calling Transition with a
// single change for creation/removal, testDecomposeEntry allows one to
// decompose an Entry into a single transition per node. It can perform
// decomposition for both creation and removal transitions.
func testDecomposeEntry(path string, entry *Entry, creation bool) []*Change {
	// Create a decomposer for creations.
	decomposer := &testEntryDecomposer{creation: creation}

	// Have it perform decomposition.
	decomposer.decompose(path, entry)

	// Return the relevant transitions.
	return decomposer.transitions
}

// testProvider is an implementation of the Provider interfaces for tests. It
// loads file data from a content map.
type testProvider struct {
	// servingRoot is the temporary directory where "staged" files are served
	// from.
	servingRoot string
	// contentMap is a map from path to file content.
	contentMap map[string][]byte
	// hasher is the hasher to use when verifying content.
	hasher hash.Hash
}

// newTestProvider creates a new instance of testProvider with the specified
// content map.
func newTestProvider(contentMap map[string][]byte, hasher hash.Hash) (*testProvider, error) {
	// Create a temporary directory for serving files.
	servingRoot, err := ioutil.TempDir("", "mutagen_provide_root")
	if err != nil {
		return nil, errors.Wrap(err, "unable to create serving directory")
	}

	// Create the test provider.
	return &testProvider{
		servingRoot: servingRoot,
		contentMap:  contentMap,
		hasher:      hasher,
	}, nil
}

// Provide implements the Provider interface for testProvider.
func (p *testProvider) Provide(path string, entry *Entry, baseMode os.FileMode) (string, error) {
	// Ensure the entry is a file type.
	if entry.Kind != EntryKind_File {
		return "", errors.New("invalid entry kind provision requested")
	}

	// Grab the content for this path.
	content, ok := p.contentMap[path]
	if !ok {
		return "", errors.New("unable to find content for path")
	}

	// Ensure it matches the requested hash.
	p.hasher.Reset()
	p.hasher.Write(content)
	if !bytes.Equal(entry.Digest, p.hasher.Sum(nil)) {
		return "", errors.New("requested entry digest does not match expected")
	}

	// Create a temporary file in the serving root.
	temporaryFile, err := ioutil.TempFile(p.servingRoot, "mutagen_provide")
	if err != nil {
		return "", errors.Wrap(err, "unable to create temporary file")
	}

	// Write content.
	_, err = temporaryFile.Write(content)
	temporaryFile.Close()
	if err != nil {
		os.Remove(temporaryFile.Name())
		return "", errors.Wrap(err, "unable to write file contents")
	}

	// Compute the file mode.
	mode := baseMode
	if mode == 0 {
		mode = ProviderBaseMode
	}
	if entry.Executable {
		mode |= UserExecutablePermission
	} else {
		mode &^= AnyExecutablePermission
	}

	// Set the file mode.
	if err := os.Chmod(temporaryFile.Name(), mode); err != nil {
		os.Remove(temporaryFile.Name())
		return "", errors.Wrap(err, "unable to set file mode")
	}

	// Success.
	return temporaryFile.Name(), nil
}

// finalize removes the temporary serving directory underlying the testProvider.
func (p *testProvider) finalize() error {
	return os.RemoveAll(p.servingRoot)
}

// testTransitionCreate creates test content on disk using Transition based on
// the specified entry and content map. It can optionally decompose the entry
// into individual node creations to stress-test Transition.
func testTransitionCreate(entry *Entry, contentMap map[string][]byte, decompose bool) (string, string, error) {
	// Create temporary directory to act as the parent of our root.
	parent, err := ioutil.TempDir("", "mutagen_simulated")
	if err != nil {
		return "", "", errors.Wrap(err, "unable to create temporary root parent")
	}

	// Compute the path to the root.
	root := filepath.Join(parent, "root")

	// Set up the creation transitions.
	var transitions []*Change
	if decompose {
		transitions = testDecomposeEntry("", entry, true)
	} else {
		transitions = []*Change{{New: entry}}
	}

	// Create a provider and ensure its cleanup.
	provider, err := newTestProvider(contentMap, newTestHasher())
	if err != nil {
		return "", "", errors.Wrap(err, "unable to create test provider")
	}
	defer provider.finalize()

	// Perform the creation transition.
	if entries, problems := Transition(root, transitions, nil, SymlinkMode_Sane, provider); len(problems) != 0 {
		os.RemoveAll(parent)
		return "", "", errors.New("problems occurred during creation transition")
	} else if len(entries) != len(transitions) {
		os.RemoveAll(parent)
		return "", "", errors.New("unexpected number of entries returned from creation transition")
	} else {
		for e, entry := range entries {
			if !entry.Equal(transitions[e].New) {
				os.RemoveAll(parent)
				return "", "", errors.New("created entry does not match expected")
			}
		}
	}

	// Success.
	return root, parent, nil
}

// testTransitionRemove removes test content from disk using Transition based on
// the specified entry. It can optionally decompose the entry into individual
// node removals to stress-test Transition.
func testTransitionRemove(root string, expected *Entry, cache *Cache, symlinkMode SymlinkMode, decompose bool) error {
	// Set up the removal transitions.
	var transitions []*Change
	if decompose {
		transitions = testDecomposeEntry("", expected, false)
	} else {
		transitions = []*Change{{Old: expected}}
	}

	// Perform the removal transition.
	if entries, problems := Transition(root, transitions, cache, symlinkMode, nil); len(problems) != 0 {
		return errors.New("problems occurred during removal transition")
	} else if len(entries) != len(transitions) {
		return errors.New("unexpected number of entries returned from removal transition")
	} else {
		for _, entry := range entries {
			if entry != nil {
				return errors.New("post-removal entry non-nil")
			}
		}
	}

	// Success.
	return nil
}

func testTransitionCycle(entry *Entry, contentMap map[string][]byte, decompose bool) error {
	// Create test content on disk and defer its removal. This will exercise
	// the creation portion of Transition.
	root, parent, err := testTransitionCreate(entry, contentMap, decompose)
	if err != nil {
		return errors.Wrap(err, "unable to create test content")
	}
	defer os.RemoveAll(parent)

	// Compute the expected entry. If we're on a system that doesn't preserve
	// executability, then strip executability from the expected value.
	expected := entry
	if !filesystem.PreservesExecutability {
		expected = StripExecutability(expected)
	}

	// Create a hasher.
	hasher := newTestHasher()

	// Perform a scan.
	snapshot, cache, err := Scan(root, hasher, nil, nil, SymlinkMode_Sane)
	if err != nil {
		return errors.Wrap(err, "unable to perform scan")
	} else if cache == nil {
		return errors.New("nil cache returned")
	} else if !snapshot.Equal(expected) {
		return errors.New("snapshot not equal to expected")
	}

	// Remove the test content. This will exercise the removal portion of
	// Transition.
	if err := testTransitionRemove(root, expected, cache, SymlinkMode_Sane, decompose); err != nil {
		return errors.Wrap(err, "unable to remove test content")
	}

	// Success.
	return nil
}

func TestTransitionNilRoot(t *testing.T) {
	// Test the nominal case.
	if err := testTransitionCycle(testNilEntry, nil, false); err != nil {
		t.Error("transition cycle failed:", err)
	}

	// Test the decomposed case.
	if err := testTransitionCycle(testNilEntry, nil, true); err != nil {
		t.Error("decomposed transition cycle failed:", err)
	}
}

func TestTransitionFile1Root(t *testing.T) {
	// Test the nominal case.
	if err := testTransitionCycle(testFile1Entry, testFile1ContentMap, false); err != nil {
		t.Error("transition cycle failed:", err)
	}

	// Test the decomposed case.
	if err := testTransitionCycle(testFile1Entry, testFile1ContentMap, true); err != nil {
		t.Error("decomposed transition cycle failed:", err)
	}
}

func TestTransitionFile2Root(t *testing.T) {
	// Test the nominal case.
	if err := testTransitionCycle(testFile2Entry, testFile2ContentMap, false); err != nil {
		t.Error("transition cycle failed:", err)
	}

	// Test the decomposed case.
	if err := testTransitionCycle(testFile2Entry, testFile2ContentMap, true); err != nil {
		t.Error("decomposed transition cycle failed:", err)
	}
}

func TestTransitionFile3Root(t *testing.T) {
	// Test the nominal case.
	if err := testTransitionCycle(testFile3Entry, testFile3ContentMap, false); err != nil {
		t.Error("transition cycle failed:", err)
	}

	// Test the decomposed case.
	if err := testTransitionCycle(testFile3Entry, testFile3ContentMap, true); err != nil {
		t.Error("decomposed transition cycle failed:", err)
	}
}

func TestTransitionDirectory1Root(t *testing.T) {
	// Test the nominal case.
	if err := testTransitionCycle(testDirectory1Entry, testDirectory1ContentMap, false); err != nil {
		t.Error("transition cycle failed:", err)
	}

	// Test the decomposed case.
	if err := testTransitionCycle(testDirectory1Entry, testDirectory1ContentMap, true); err != nil {
		t.Error("decomposed transition cycle failed:", err)
	}
}

func TestTransitionDirectory2Root(t *testing.T) {
	// Test the nominal case.
	if err := testTransitionCycle(testDirectory2Entry, testDirectory2ContentMap, false); err != nil {
		t.Error("transition cycle failed:", err)
	}

	// Test the decomposed case.
	if err := testTransitionCycle(testDirectory2Entry, testDirectory2ContentMap, true); err != nil {
		t.Error("decomposed transition cycle failed:", err)
	}
}

func TestTransitionDirectory3Root(t *testing.T) {
	// Test the nominal case.
	if err := testTransitionCycle(testDirectory3Entry, testDirectory3ContentMap, false); err != nil {
		t.Error("transition cycle failed:", err)
	}

	// Test the decomposed case.
	if err := testTransitionCycle(testDirectory3Entry, testDirectory3ContentMap, true); err != nil {
		t.Error("decomposed transition cycle failed:", err)
	}
}

func TestTransitionCaseConflict(t *testing.T) {
	// Determine whether or not we expect case conflicts.
	// HACK: We actually ought to be determining this based on the filesystem
	// being used, but it's a sufficient test mechanism for now.
	expectCaseConflict := runtime.GOOS == "windows" || runtime.GOOS == "darwin"

	// Check for case conflicts in the nominal case.
	err := testTransitionCycle(testDirectoryWithCaseConflict, testDirectoryWithCaseConflictContentMap, false)
	if expectCaseConflict && err == nil {
		t.Error("expected case conflict")
	} else if !expectCaseConflict && err != nil {
		t.Error("unexpected case conflict")
	}

	// Check for case conflicts in the decomposed case.
	err = testTransitionCycle(testDirectoryWithCaseConflict, testDirectoryWithCaseConflictContentMap, true)
	if expectCaseConflict && err == nil {
		t.Error("expected decomposed case conflict")
	} else if !expectCaseConflict && err != nil {
		t.Error("unexpected decomposed case conflict")
	}
}
