package sync

const (
	testFile1Contents = "Hello, world!"
	testFile2Contents = "#!/bin/bash\necho 'Hello, world!'"
	testFile3Contents = "Something else"
)

var testFile1ContentsSHA1 = []byte{
	0x94, 0x3a, 0x70, 0x2d, 0x06, 0xf3, 0x45, 0x99, 0xae, 0xe1,
	0xf8, 0xda, 0x8e, 0xf9, 0xf7, 0x29, 0x60, 0x31, 0xd6, 0x99,
}

var testFile2ContentsSHA1 = []byte{
	0xe5, 0x38, 0x33, 0x8f, 0xc4, 0xe9, 0x98, 0x3b, 0xf4, 0x8e,
	0x04, 0xba, 0x41, 0x59, 0x25, 0xc0, 0x0b, 0x33, 0xe5, 0xcb,
}

var testFile3ContentsSHA1 = []byte{
	0x48, 0xf8, 0x8a, 0xc3, 0x22, 0xa0, 0x66, 0x76, 0x82, 0xd3,
	0x17, 0x1b, 0x3e, 0x51, 0x9a, 0xb7, 0x36, 0x06, 0x59, 0x96,
}

var testNilEntry *Entry

// testDirectory1Entry is a sample directory entry for use in testing. It has
// the following structure:
//
//   /
//     empty directory/
//     directory/
//       subdirectory/
//       subfile (file 3)
//       another symlink (=> ../executable file)
//     second directory/
//       subfile.exe (file 3)
//     file (file 1)
//     executable file (file 2, executable)
//     symlink (=> directory/subfile)
var testDirectory1Entry = &Entry{
	Kind: EntryKind_Directory,
	Contents: map[string]*Entry{
		"empty directory": {
			Kind: EntryKind_Directory,
		},
		"directory": {
			Kind: EntryKind_Directory,
			Contents: map[string]*Entry{
				"subdirectory": {
					Kind: EntryKind_Directory,
				},
				"subfile": {
					Kind:   EntryKind_File,
					Digest: testFile3ContentsSHA1,
				},
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
					Kind:   EntryKind_File,
					Digest: testFile3ContentsSHA1,
				},
			},
		},
		"file": {
			Kind:   EntryKind_File,
			Digest: testFile1ContentsSHA1,
		},
		"executable file": {
			Kind:       EntryKind_File,
			Executable: true,
			Digest:     testFile2ContentsSHA1,
		},
		"symlink": {
			Kind:   EntryKind_Symlink,
			Target: "directory/subfile",
		},
	},
}

// testDirectory2Entry is a sample directory entry for use in testing. It has
// the following structure:
//
//   /
//     empty directory/
//       new subfile (file 3)
//     renamed directory/
//       subdirectory/
//       subfile (file 3)
//       another symlink (=> ../executable file)
//     second directory/
//       subfile.exe (file 3, executable)
//     renamed file (file 1)
//     executable file (file 2, executable)
//     new symlink (=> renamed directory/subfile)
var testDirectory2Entry = &Entry{
	Kind: EntryKind_Directory,
	Contents: map[string]*Entry{
		"empty directory": {
			Kind: EntryKind_Directory,
			Contents: map[string]*Entry{
				"new subfile": {
					Kind:   EntryKind_File,
					Digest: testFile3ContentsSHA1,
				},
			},
		},
		"renamed directory": {
			Kind: EntryKind_Directory,
			Contents: map[string]*Entry{
				"subdirectory": {
					Kind: EntryKind_Directory,
				},
				"subfile": {
					Kind:   EntryKind_File,
					Digest: testFile3ContentsSHA1,
				},
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
		"renamed_file": {
			Kind:   EntryKind_File,
			Digest: testFile1ContentsSHA1,
		},
		"executable file": {
			Kind:       EntryKind_File,
			Executable: true,
			Digest:     testFile2ContentsSHA1,
		},
		"new symlink": {
			Kind:   EntryKind_Symlink,
			Target: "renamed directory/subfile",
		},
	},
}

// testDirectory3Entry is a sample directory entry for use in testing. It is a
// subentry of testDirectory2Entry. It has the following structure:
//
//   /
//     empty directory/
//       new subfile (file 3)
//     renamed directory/
//       subdirectory/
//       subfile (file 3)
//       another symlink (=> ../executable file)
//     second directory/
//       subfile.exe (file 3, executable)
//     executable file (file 2, executable)
//     new symlink (=> renamed directory/subfile)
var testDirectory3Entry = &Entry{
	Kind: EntryKind_Directory,
	Contents: map[string]*Entry{
		"empty directory": {
			Kind: EntryKind_Directory,
			Contents: map[string]*Entry{
				"new subfile": {
					Kind:   EntryKind_File,
					Digest: testFile3ContentsSHA1,
				},
			},
		},
		"renamed directory": {
			Kind: EntryKind_Directory,
			Contents: map[string]*Entry{
				"subdirectory": {
					Kind: EntryKind_Directory,
				},
				"subfile": {
					Kind:   EntryKind_File,
					Digest: testFile3ContentsSHA1,
				},
				"another symlink": {
					Kind:   EntryKind_Symlink,
					Target: "../executable file",
				},
			},
		},
		"executable file": {
			Kind:       EntryKind_File,
			Executable: true,
			Digest:     testFile2ContentsSHA1,
		},
		"new symlink": {
			Kind:   EntryKind_Symlink,
			Target: "renamed directory/subfile",
		},
	},
}

var testFileEntry = &Entry{
	Kind:   EntryKind_File,
	Digest: testFile1ContentsSHA1,
}

var testSymlinkEntry = &Entry{
	Kind:   EntryKind_Symlink,
	Target: "file",
}

func createTestDirectoryRoot() (string, error) {
	// TODO: Implement.
	return "", nil
}

func createTestFileRoot() (string, error) {
	// TODO: Implement.
	return "", nil
}
