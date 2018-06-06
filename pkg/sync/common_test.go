package sync

import (
	"bytes"
	"crypto/sha1"
	"hash"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

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

var testDirectory1ContentMap = map[string][]byte{
	"directory/subfile":            testFile3Contents,
	"second directory/subfile.exe": testFile3Contents,
	"file":            testFile1Contents,
	"executable file": testFile2Contents,
}

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

var testDirectory2ContentMap = map[string][]byte{
	"empty directory/new subfile":  testFile3Contents,
	"renamed directory/subfile":    testFile3Contents,
	"second directory/subfile.exe": testFile3Contents,
	"renamed_file":                 testFile1Contents,
	"executable file":              testFile2Contents,
}

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

var testDirectory3ContentMap = map[string][]byte{
	"empty directory/new subfile":  testFile3Contents,
	"renamed directory/subfile":    testFile3Contents,
	"second directory/subfile.exe": testFile3Contents,
	"executable file":              testFile2Contents,
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

type testProvider struct {
	servingRoot string
	contentMap  map[string][]byte
}

func newTestProvider(contentMap map[string][]byte) (*testProvider, error) {
	// Create a temporary directory for serving files.
	servingRoot, err := ioutil.TempDir("", "mutagen_provide_root")
	if err != nil {
		return nil, errors.Wrap(err, "unable to create serving directory")
	}

	// Create the test provider.
	return &testProvider{
		servingRoot: servingRoot,
		contentMap:  contentMap,
	}, nil
}

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
	contentHash := sha1.Sum(content)
	if !bytes.Equal(entry.Digest, contentHash[:]) {
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

func (p *testProvider) Finalize() error {
	return os.RemoveAll(p.servingRoot)
}

func createTestContentOnDisk(entry *Entry, contentMap map[string][]byte) (string, string, error) {
	// Create a provider and ensure its cleanup.
	provider, err := newTestProvider(contentMap)
	if err != nil {
		return "", "", errors.Wrap(err, "unable to create test provider")
	}
	defer provider.Finalize()

	// Create temporary directory to act as the parent of our root and defer its
	// cleanup.
	parent, err := ioutil.TempDir("", "mutagen_simulated")
	if err != nil {
		return "", "", errors.Wrap(err, "unable to create temporary root parent")
	}

	// Compute the path to the root.
	root := filepath.Join(parent, "root")

	// Set up transitions to create the specified entry at the root.
	transitions := []*Change{{New: entry}}

	// Create an empty cache for the transition. This is fine since we're only
	// doing creations and don't need the cache.
	cache := &Cache{}

	// Perform the creation transition.
	if entries, problems := Transition(root, transitions, cache, provider); len(problems) != 0 {
		os.RemoveAll(parent)
		return "", "", errors.New("problems occurred during creation transition")
	} else if len(entries) != 1 {
		os.RemoveAll(parent)
		return "", "", errors.New("unexpected number of entries returned from creation transition")
	} else if !entries[0].Equal(entry) {
		os.RemoveAll(parent)
		return "", "", errors.New("created entry does not match expected")
	}

	// Success.
	return root, parent, nil
}
