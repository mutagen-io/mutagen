package sync

import (
	"crypto/sha1"
	"hash"
	"os"
	"runtime"
)

type temporaryDirectory struct {
	name string
	path string
}

// testingTemporaryDirectories returns the list of paths that should be used as
// temporary directories in testing.
func testingTemporaryDirectories() []temporaryDirectory {
	// Create the initial result with the default temporary directory.
	results := []temporaryDirectory{
		{"OS", ""},
	}

	// If we're not on Windows, and there's a FAT32 root to test in, then add
	// that. The reason we skip this on Windows is because FAT32 on Windows
	// doesn't allow for symlinks, which we attempt to create extensively in our
	// tests. It's not that Mutagen doesn't behave correctly in this case (it
	// will still safely synchronize and just indicate an inability to propagate
	// symlinks in that case), but our tests assume that we can freely create
	// symlinks. We still use this partition in other places on Windows though,
	// e.g. in filesystem tests.
	if runtime.GOOS != "windows" {
		if root := os.Getenv("MUTAGEN_TEST_FAT32_ROOT"); root != "" {
			results = append(results, temporaryDirectory{"FAT32", root})
		}
	}

	// If there's an HFS+ root to test in, then add that.
	if root := os.Getenv("MUTAGEN_TEST_HFS_ROOT"); root != "" {
		results = append(results, temporaryDirectory{"HFS+", root})
	}

	// If there's an APFS root to test in, then add that.
	if root := os.Getenv("MUTAGEN_TEST_APFS_ROOT"); root != "" {
		results = append(results, temporaryDirectory{"APFS", root})
	}

	// Done.
	return results
}

func newTestHasher() hash.Hash {
	return sha1.New()
}

var testFile1Contents = []byte("Hello, world!")

var testFile1ContentsSHA1 = []byte{
	0x94, 0x3a, 0x70, 0x2d, 0x06, 0xf3, 0x45, 0x99, 0xae, 0xe1,
	0xf8, 0xda, 0x8e, 0xf9, 0xf7, 0x29, 0x60, 0x31, 0xd6, 0x99,
}

var testFile2Contents = []byte("#!/bin/bash\necho 'Hello, world!'")

var testFile2ContentsSHA1 = []byte{
	0xc8, 0x5f, 0x28, 0x3f, 0xa0, 0xb4, 0x2d, 0xf3, 0x28, 0xf4,
	0x10, 0xd4, 0xf3, 0x64, 0x4c, 0x78, 0x0b, 0x30, 0xda, 0x68,
}

var testFile3Contents = []byte("Something else")

var testFile3ContentsSHA1 = []byte{
	0x48, 0xf8, 0x8a, 0xc3, 0x22, 0xa0, 0x66, 0x76, 0x82, 0xd3,
	0x17, 0x1b, 0x3e, 0x51, 0x9a, 0xb7, 0x36, 0x06, 0x59, 0x96,
}

var testNilEntry *Entry

var testFile1Entry = &Entry{
	Kind:   EntryKind_File,
	Digest: testFile1ContentsSHA1,
}

var testFile1ContentMap = map[string][]byte{
	"": testFile1Contents,
}

var testFile2Entry = &Entry{
	Kind:       EntryKind_File,
	Digest:     testFile2ContentsSHA1,
	Executable: true,
}

var testFile2ContentMap = map[string][]byte{
	"": testFile2Contents,
}

var testFile3Entry = &Entry{
	Kind:   EntryKind_File,
	Digest: testFile3ContentsSHA1,
}

var testFile3ContentMap = map[string][]byte{
	"": testFile3Contents,
}

var testEmptyDirectory = &Entry{}

var testDirectory1Entry = &Entry{
	Kind: EntryKind_Directory,
	Contents: map[string]*Entry{
		"empty dir\xc3\xa9ctory": {
			Kind: EntryKind_Directory,
		},
		"directory": {
			Kind: EntryKind_Directory,
			Contents: map[string]*Entry{
				"subdirectory": {
					Kind: EntryKind_Directory,
				},
				"subfile": testFile3Entry,
				"another symlink": {
					Kind:   EntryKind_Symlink,
					Target: "../executable file",
				},
			},
		},
		"second directory": {
			Kind: EntryKind_Directory,
			Contents: map[string]*Entry{
				"subfile.exe": testFile3Entry,
			},
		},
		"file":            testFile1Entry,
		"executable file": testFile2Entry,
		"symlink": {
			Kind:   EntryKind_Symlink,
			Target: "directory/subfile",
		},
	},
}

var testDirectory1ContentMap = map[string][]byte{
	"directory/subfile":            testFile3Contents,
	"second directory/subfile.exe": testFile3Contents,
	"file":                         testFile1Contents,
	"executable file":              testFile2Contents,
}

var testDirectory2Entry = &Entry{
	Kind: EntryKind_Directory,
	Contents: map[string]*Entry{
		"empty dir\xc3\xa9ctory": {
			Kind: EntryKind_Directory,
			Contents: map[string]*Entry{
				"new subfile": testFile3Entry,
			},
		},
		"renamed directory": {
			Kind: EntryKind_Directory,
			Contents: map[string]*Entry{
				"subdirectory": {
					Kind: EntryKind_Directory,
				},
				"subfile": testFile3Entry,
				"another symlink": {
					Kind:   EntryKind_Symlink,
					Target: "../executable file",
				},
			},
		},
		"second directory": {
			Kind: EntryKind_Directory,
			Contents: map[string]*Entry{
				"subfile.exe": {
					Kind:       EntryKind_File,
					Executable: true,
					Digest:     testFile3ContentsSHA1,
				},
			},
		},
		"renamed_file":    testFile1Entry,
		"executable file": testFile2Entry,
		"new symlink": {
			Kind:   EntryKind_Symlink,
			Target: "renamed directory/subfile",
		},
	},
}

var testDirectory2ContentMap = map[string][]byte{
	"empty dir\xc3\xa9ctory/new subfile": testFile3Contents,
	"renamed directory/subfile":          testFile3Contents,
	"second directory/subfile.exe":       testFile3Contents,
	"renamed_file":                       testFile1Contents,
	"executable file":                    testFile2Contents,
}

var testDirectory3Entry = &Entry{
	Kind: EntryKind_Directory,
	Contents: map[string]*Entry{
		"empty dir\xc3\xa9ctory": {
			Kind: EntryKind_Directory,
			Contents: map[string]*Entry{
				"new subfile": testFile3Entry,
			},
		},
		"renamed directory": {
			Kind: EntryKind_Directory,
			Contents: map[string]*Entry{
				"subdirectory": {
					Kind: EntryKind_Directory,
				},
				"subfile": testFile3Entry,
				"another symlink": {
					Kind:   EntryKind_Symlink,
					Target: "../executable file",
				},
			},
		},
		"executable file": testFile2Entry,
		"new symlink": {
			Kind:   EntryKind_Symlink,
			Target: "renamed directory/subfile",
		},
	},
}

var testDirectory3ContentMap = map[string][]byte{
	"empty dir\xc3\xa9ctory/new subfile": testFile3Contents,
	"renamed directory/subfile":          testFile3Contents,
	"second directory/subfile.exe":       testFile3Contents,
	"executable file":                    testFile2Contents,
}

var testDirectoryWithCaseConflict = &Entry{
	Kind: EntryKind_Directory,
	Contents: map[string]*Entry{
		"FileName": testFile1Entry,
		"FILENAME": testFile3Entry,
	},
}

var testDirectoryWithCaseConflictContentMap = map[string][]byte{
	"FileName": testFile1Contents,
	"FILENAME": testFile3Contents,
}

var testSymlinkEntry = &Entry{
	Kind:   EntryKind_Symlink,
	Target: "file",
}

var testDirectoryWithSaneSymlink = &Entry{
	Kind: EntryKind_Directory,
	Contents: map[string]*Entry{
		"sane symlink": {
			Kind:   EntryKind_Symlink,
			Target: "neighboring file",
		},
	},
}

var testDirectoryWithInvalidSymlink = &Entry{
	Kind: EntryKind_Directory,
	Contents: map[string]*Entry{
		"invalid-symlink": {
			Kind:   EntryKind_Symlink,
			Target: "neighboring:file",
		},
	},
}

var testDirectoryWithEscapingSymlink = &Entry{
	Kind: EntryKind_Directory,
	Contents: map[string]*Entry{
		"escaping_symlink": {
			Kind:   EntryKind_Symlink,
			Target: "../parent neighbor",
		},
	},
}

var testDirectoryWithAbsoluteSymlink = &Entry{
	Kind: EntryKind_Directory,
	Contents: map[string]*Entry{
		"absolute symlink": {
			Kind:   EntryKind_Symlink,
			Target: "/path/to/neighboring file",
		},
	},
}
