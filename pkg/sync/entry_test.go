package sync

import (
	"testing"
)

func TestEntryNilValid(t *testing.T) {
	if err := testNilEntry.EnsureValid(); err != nil {
		t.Fatal("nil entry considered invalid:", err)
	}
}

func TestEntryDirectoryExecutableInvalid(t *testing.T) {
	directory := &Entry{
		Kind:       EntryKind_Directory,
		Executable: true,
	}
	if directory.EnsureValid() == nil {
		t.Fatal("directory with executability set considered valid")
	}
}

func TestEntryDirectoryDigestInvalid(t *testing.T) {
	directory := &Entry{
		Kind:   EntryKind_Directory,
		Digest: []byte{0},
	}
	if directory.EnsureValid() == nil {
		t.Fatal("directory with digest set considered valid")
	}
}

func TestEntryDirectoryTargetInvalid(t *testing.T) {
	directory := &Entry{
		Kind:   EntryKind_Directory,
		Target: "file",
	}
	if directory.EnsureValid() == nil {
		t.Fatal("directory with target set considered valid")
	}
}

func TestEntryDirectoryEmptyContentNameInvalid(t *testing.T) {
	directory := &Entry{
		Kind: EntryKind_Directory,
		Contents: map[string]*Entry{
			"": testFile1Entry,
		},
	}
	if directory.EnsureValid() == nil {
		t.Fatal("directory with empty content name considered valid")
	}
}

func TestEntryDirectoryContentNameWithDotInvalid(t *testing.T) {
	directory := &Entry{
		Kind: EntryKind_Directory,
		Contents: map[string]*Entry{
			".": testFile1Entry,
		},
	}
	if directory.EnsureValid() == nil {
		t.Fatal("directory with empty content name considered valid")
	}
}

func TestEntryDirectoryContentNameWithDotDotInvalid(t *testing.T) {
	directory := &Entry{
		Kind: EntryKind_Directory,
		Contents: map[string]*Entry{
			"..": testFile1Entry,
		},
	}
	if directory.EnsureValid() == nil {
		t.Fatal("directory with empty content name considered valid")
	}
}

func TestEntryDirectoryContentNameWithPathSeparatorInvalid(t *testing.T) {
	directory := &Entry{
		Kind: EntryKind_Directory,
		Contents: map[string]*Entry{
			"na/me": testFile1Entry,
		},
	}
	if directory.EnsureValid() == nil {
		t.Fatal("directory with path separator in content name considered valid")
	}
}

func TestEntryDirectoryNilContentInvalid(t *testing.T) {
	directory := &Entry{
		Kind: EntryKind_Directory,
		Contents: map[string]*Entry{
			"file": nil,
		},
	}
	if directory.EnsureValid() == nil {
		t.Fatal("directory with nil content considered valid")
	}
}

func TestEntryDirectoryInvalidContentInvalid(t *testing.T) {
	directory := &Entry{
		Kind: EntryKind_Directory,
		Contents: map[string]*Entry{
			"file": {Kind: EntryKind_File},
		},
	}
	if directory.EnsureValid() == nil {
		t.Fatal("directory with invalid content considered valid")
	}
}

func TestEntryDirectoryValid(t *testing.T) {
	if err := testDirectory1Entry.EnsureValid(); err != nil {
		t.Fatal("valid directory considered invalid:", err)
	}
}

func TestEntryFileContentsInvalid(t *testing.T) {
	file := &Entry{
		Kind: EntryKind_File,
		Contents: map[string]*Entry{
			"file": testFile1Entry,
		},
	}
	if file.EnsureValid() == nil {
		t.Fatal("file with directory content considered valid")
	}
}

func TestEntryFileTargetInvalid(t *testing.T) {
	file := &Entry{
		Kind:   EntryKind_File,
		Digest: []byte{0},
		Target: "file",
	}
	if file.EnsureValid() == nil {
		t.Fatal("file with target set considered valid")
	}
}

func TestEntryFileNilDigestInvalid(t *testing.T) {
	file := &Entry{
		Kind: EntryKind_File,
	}
	if file.EnsureValid() == nil {
		t.Fatal("file with nil digest considered valid")
	}
}

func TestEntryFileEmptyDigestInvalid(t *testing.T) {
	file := &Entry{
		Kind:   EntryKind_File,
		Digest: []byte{},
	}
	if file.EnsureValid() == nil {
		t.Fatal("file with empty digest considered valid")
	}
}

func TestEntryFileValid(t *testing.T) {
	if err := testFile1Entry.EnsureValid(); err != nil {
		t.Fatal("valid file considered invalid:", err)
	}
}

func TestEntrySymlinkExecutableInvalid(t *testing.T) {
	symlink := &Entry{
		Kind:       EntryKind_Symlink,
		Target:     "file",
		Executable: true,
	}
	if symlink.EnsureValid() == nil {
		t.Fatal("symlink with executability set considered valid")
	}
}

func TestEntrySymlinkDigestInvalid(t *testing.T) {
	symlink := &Entry{
		Kind:   EntryKind_Symlink,
		Target: "file",
		Digest: []byte{0},
	}
	if symlink.EnsureValid() == nil {
		t.Fatal("symlink with digest set considered valid")
	}
}

func TestEntrySymlinkContentsInvalid(t *testing.T) {
	symlink := &Entry{
		Kind:   EntryKind_Symlink,
		Target: "file",
		Contents: map[string]*Entry{
			"file": testFile1Entry,
		},
	}
	if symlink.EnsureValid() == nil {
		t.Fatal("symlink with directory content considered valid")
	}
}

func TestEntrySymlinkTargetEmptyInvalid(t *testing.T) {
	symlink := &Entry{
		Kind:   EntryKind_Symlink,
		Target: "",
	}
	if symlink.EnsureValid() == nil {
		t.Fatal("symlink with empty target considered valid")
	}
}

func TestEntrySymlinkValid(t *testing.T) {
	if err := testSymlinkEntry.EnsureValid(); err != nil {
		t.Fatal("valid symlink considered invalid:", err)
	}
}

func TestEntryInvalidKindInvalid(t *testing.T) {
	entry := &Entry{Kind: (EntryKind_Symlink + 1)}
	if entry.EnsureValid() == nil {
		t.Fatal("entry with invalid kind considered valid")
	}
}

func TestEntryWalk(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		entry            *Entry
		expectedContents map[string]*Entry
	}{
		{nil, map[string]*Entry{"": nil}},
		{testFile1Entry, map[string]*Entry{"": testFile1Entry}},
		{testDirectoryWithCaseConflict, map[string]*Entry{
			"":         testDirectoryWithCaseConflict,
			"FileName": testFile1Entry,
			"FILENAME": testFile3Entry,
		}},
		{testDirectory3Entry, map[string]*Entry{
			"":                                   testDirectory3Entry,
			"empty dir\xc3\xa9ctory":             testDirectory3Entry.Contents["empty dir\xc3\xa9ctory"],
			"empty dir\xc3\xa9ctory/new subfile": testFile3Entry,
			"renamed directory":                  testDirectory3Entry.Contents["renamed directory"],
			"renamed directory/subdirectory":     testDirectory3Entry.Contents["renamed directory"].Contents["subdirectory"],
			"renamed directory/subfile":          testFile3Entry,
			"renamed directory/another symlink":  testDirectory3Entry.Contents["renamed directory"].Contents["another symlink"],
			"executable file":                    testFile2Entry,
			"new symlink":                        testDirectory3Entry.Contents["new symlink"],
		}},
	}

	// Process test cases.
	for _, testCase := range testCases {
		// Perform walking to extract contents.
		contents := make(map[string]*Entry)
		testCase.entry.walk("", func(path string, entry *Entry) {
			contents[path] = entry
		})

		// Compare content lengths.
		if len(contents) != len(testCase.expectedContents) {
			t.Error(
				"content length does not match expected:",
				len(contents),
				"!=",
				len(testCase.expectedContents),
			)
			continue
		}

		// Compare contents.
		for path, expectedEntry := range testCase.expectedContents {
			if entry, ok := contents[path]; !ok {
				t.Error("unable to find expected content path")
			} else if !entry.Equal(expectedEntry) {
				t.Error("content entry not equal to expected")
			}
		}
	}
}

func TestEntryCountNil(t *testing.T) {
	if count := testNilEntry.Count(); count != 0 {
		t.Error("zero-entry hierarchy reported incorrect count:", count)
	}
}

func TestEntryCountSingle(t *testing.T) {
	if count := testFile1Entry.Count(); count != 1 {
		t.Error("single-entry hierarchy reported incorrect count:", count)
	}
}

func TestEntryCountHierarchy(t *testing.T) {
	if count := testDirectory1Entry.Count(); count != 11 {
		t.Error("multi-entry hierarchy reported incorrect count:", count, "!=", 11)
	}
}

func TestEntryNilNilEqualShallow(t *testing.T) {
	if !testNilEntry.equalShallow(testNilEntry) {
		t.Error("two nil entries not considered shallow equal")
	}
}

func TestEntryNilNonNilNotEqualShallow(t *testing.T) {
	if testNilEntry.equalShallow(testFile1Entry) {
		t.Error("nil and non-nil entries considered shallow equal")
	}
}

func TestEntrySameDirectoryEqualShallow(t *testing.T) {
	if !testDirectory1Entry.equalShallow(testDirectory1Entry) {
		t.Error("identical directories not considered shallow equal")
	}
}

func TestEntrySameFileEqualShallow(t *testing.T) {
	if !testFile1Entry.equalShallow(testFile1Entry) {
		t.Error("identical files not considered shallow equal")
	}
}

func TestEntrySameSymlinkEqualShallow(t *testing.T) {
	if !testSymlinkEntry.equalShallow(testSymlinkEntry) {
		t.Error("identical symlinks not considered shallow equal")
	}
}

func TestEntrySymlinkFileNotEqualShallow(t *testing.T) {
	if testSymlinkEntry.equalShallow(testFile1Entry) {
		t.Error("symlink and file considered shallow equal")
	}
}

func TestDifferentDirectoriesEqualShallow(t *testing.T) {
	if !testDirectory1Entry.equalShallow(testDirectory2Entry) {
		t.Error("different directories not considered shallow equal")
	}
}

func TestEntryNilNilEqual(t *testing.T) {
	if !testNilEntry.Equal(testNilEntry) {
		t.Error("two nil entries not considered equal")
	}
}

func TestEntryEmptyDirectoriesEqual(t *testing.T) {
	emptyDirectory := &Entry{
		Kind:     EntryKind_Directory,
		Contents: make(map[string]*Entry),
	}
	emptyDirectoryNilContent := &Entry{
		Kind: EntryKind_Directory,
	}
	if !emptyDirectory.Equal(emptyDirectoryNilContent) {
		t.Error("two nil entries not considered equal (empty-to-nil)")
	}
	if !emptyDirectoryNilContent.Equal(emptyDirectory) {
		t.Error("two nil entries not considered equal (nil-to-empty)")
	}
	if !emptyDirectoryNilContent.Equal(emptyDirectoryNilContent) {
		t.Error("two nil entries not considered equal (nil-to-nil)")
	}
	if !emptyDirectory.Equal(emptyDirectory) {
		t.Error("two nil entries not considered equal (empty-to-empty)")
	}
}

func TestEntrySameDirectoryEqual(t *testing.T) {
	if !testDirectory1Entry.Equal(testDirectory1Entry) {
		t.Error("identical directories not considered equal")
	}
}

func TestEntrySameFileEqual(t *testing.T) {
	if !testFile1Entry.Equal(testFile1Entry) {
		t.Error("identical files not considered equal")
	}
}

func TestEntrySameSymlinkEqual(t *testing.T) {
	if !testSymlinkEntry.Equal(testSymlinkEntry) {
		t.Error("identical symlinks not considered equal")
	}
}

func TestEntrySymlinkFileNotEqual(t *testing.T) {
	if testSymlinkEntry.Equal(testFile1Entry) {
		t.Error("symlink and file considered shallow equal")
	}
}

func TestDifferentDirectoriesNotEqual(t *testing.T) {
	if testDirectory1Entry.Equal(testDirectory2Entry) {
		t.Error("directories 1 and 2 considered equal")
	}
	if testDirectory1Entry.Equal(testDirectory3Entry) {
		t.Error("directories 1 and 3 considered equal")
	}
	if testDirectory2Entry.Equal(testDirectory3Entry) {
		t.Error("directories 2 and 3 considered equal")
	}
}

func TestEntryNilCopyShallow(t *testing.T) {
	if testNilEntry.copySlim() != nil {
		t.Error("shallow copy of nil entry non-nil")
	}
}

func TestEntryDirectoryCopyShallow(t *testing.T) {
	directory := testDirectory1Entry.copySlim()
	if directory == nil {
		t.Error("shallow copy of directory returned nil")
	}
	if directory.Contents != nil {
		t.Error("shallow copy of directory has non-nil contents")
	}
	if !directory.equalShallow(testDirectory1Entry) {
		t.Error("shallow copy of directory not considered shallow equal to original")
	}
}

func TestEntryFileCopyShallow(t *testing.T) {
	file := testFile1Entry.copySlim()
	if file == nil {
		t.Error("shallow copy of file returned nil")
	}
	if !file.equalShallow(testFile1Entry) {
		t.Error("shallow copy of file not considered shallow equal to original")
	}
}

func TestEntrySymlinkCopyShallow(t *testing.T) {
	symlink := testSymlinkEntry.copySlim()
	if symlink == nil {
		t.Error("shallow copy of symlink returned nil")
	}
	if !symlink.equalShallow(testSymlinkEntry) {
		t.Error("shallow copy of symlink not considered shallow equal to original")
	}
}

func TestEntryNilCopy(t *testing.T) {
	if testNilEntry.Copy() != nil {
		t.Error("copy of nil entry non-nil")
	}
}

func TestEmptyDirectoryCopy(t *testing.T) {
	emptyDirectory := &Entry{Kind: EntryKind_Directory, Contents: make(map[string]*Entry)}
	directory := emptyDirectory.Copy()
	if directory == nil {
		t.Error("copy of empty directory returned nil")
	}
	if directory.Contents != nil {
		t.Error("copy of empty directory has non-nil contents")
	}
	if !directory.Equal(emptyDirectory) {
		t.Error("copy of empty directory not considered equal to original")
	}
}

func TestEntryDirectoryCopy(t *testing.T) {
	directory := testDirectory1Entry.Copy()
	if directory == nil {
		t.Error("copy of directory returned nil")
	}
	if !directory.Equal(testDirectory1Entry) {
		t.Error("copy of directory not considered equal to original")
	}
}

func TestEntryFileCopy(t *testing.T) {
	file := testFile1Entry.Copy()
	if file == nil {
		t.Error("copy of file returned nil")
	}
	if !file.Equal(testFile1Entry) {
		t.Error("copy of file not considered equal to original")
	}
}

func TestEntrySymlinkCopy(t *testing.T) {
	symlink := testSymlinkEntry.Copy()
	if symlink == nil {
		t.Error("copy of symlink returned nil")
	}
	if !symlink.Equal(testSymlinkEntry) {
		t.Error("copy of symlink not considered equal to original")
	}
}
