package core

// nested creates a directory entry containing an entry with the specified name.
func nested(name string, entry *Entry) *Entry {
	return &Entry{Contents: map[string]*Entry{name: entry}}
}

// executable creates a copy of an entry that's marked as executable. This
// function panics if the entry is not a file entry.
func executable(entry *Entry) *Entry {
	if entry.Kind != EntryKind_File {
		panic("entry is not a file")
	}
	return &Entry{
		Kind:       EntryKind_File,
		Digest:     entry.Digest,
		Executable: true,
	}
}

const (
	// tF1Content is the content for tF1.
	tF1Content = "first test file"
	// tF2Content is the content for tF2.
	tF2Content = "second test file"
	// tF3Content is the content for tF3 and tF3E.
	tF3Content = "#!/bin/bash\necho 'Hello, world!'"
)

// tN is a nil entry for testing.
var tN *Entry

// tF1 is a file entry for testing.
var tF1 = &Entry{Kind: EntryKind_File, Digest: testingDigest(tF1Content)}

// tF2 is an alternative file entry for testing.
var tF2 = &Entry{Kind: EntryKind_File, Digest: testingDigest(tF2Content)}

// tF3 is an alternative file entry for testing.
var tF3 = &Entry{Kind: EntryKind_File, Digest: testingDigest(tF3Content)}

// tF3E is an executable version of tF3 for testing.
var tF3E = &Entry{Kind: EntryKind_File, Digest: testingDigest(tF3Content), Executable: true}

// tSR is a symbolic link entry with a relative path for testing.
var tSR = &Entry{Kind: EntryKind_SymbolicLink, Target: "file"}

// tSA is a symbolic link entry with an absolute path for testing.
var tSA = &Entry{Kind: EntryKind_SymbolicLink, Target: "/path/file"}

// tU is an untracked entry for testing.
var tU = &Entry{Kind: EntryKind_Untracked}

// tP1 is a problematic entry for testing.
var tP1 = &Entry{Kind: EntryKind_Problematic, Problem: "something bad happened"}

// tP2 is an alternate problematic entry for testing.
var tP2 = &Entry{Kind: EntryKind_Problematic, Problem: "another bad thing happened"}

// tPInvalidUTF8 is a problematic entry indicating non-UTF-8 filename encoding.
var tPInvalidUTF8 = &Entry{Kind: EntryKind_Problematic, Problem: "non-UTF-8 filename"}

// tD0 is an empty directory entry for testing.
var tD0 = &Entry{}

// tD1 is a directory entry (containing tF1 with name "file") for testing.
var tD1 = &Entry{Contents: map[string]*Entry{"file": tF1}}

// tD2 is a directory entry (containing tF2 with name "file") for testing.
var tD2 = &Entry{Contents: map[string]*Entry{"file": tF2}}

// tD3 is a directory entry (containing tF3 with name "file") for testing.
var tD3 = &Entry{Contents: map[string]*Entry{"file": tF3}}

// tD3E is a directory entry (containing tF3E with name "file") for testing.
var tD3E = &Entry{Contents: map[string]*Entry{"file": tF3E}}

// tDCC is a directory entry with a case conflict (containing tF1 with name
// "file" and tF2 with the name "FILE") for testing.
var tDCC = &Entry{Contents: map[string]*Entry{"file": tF1, "FILE": tF2}}

// tDSR is a directory entry (containing tF1 with name "file" and tSR with name
// "symlink") for testing.
var tDSR = &Entry{Contents: map[string]*Entry{
	"file":    tF1,
	"symlink": tSR,
}}

// tDSRU is a directory entry (that mirrors tDSR with tU in place of tSR) for
// testing.
var tDSRU = &Entry{Contents: map[string]*Entry{
	"file":    tF1,
	"symlink": tU,
}}

// tDSA is a directory entry (containing tSA with name "symlink") for testing.
var tDSA = &Entry{Contents: map[string]*Entry{"symlink": tSA}}

// tDSAU is a directory entry (that mirrors tDSA with tU in place of tSA) for
// testing.
var tDSAU = &Entry{Contents: map[string]*Entry{"symlink": tU}}

// tDSAP is a directory entry (that mirrors tDSA with a problematic entry in
// place of tSA indicating an absolute symbolic link target) for testing.
var tDSAP = &Entry{Contents: map[string]*Entry{"symlink": {
	Kind: EntryKind_Problematic,
	// TODO: Is there a better way to compute this error message?
	Problem: "invalid symbolic link: target is absolute",
}}}

// tDM is a directory entry (containing multiple elements with assorted names,
// but no unsynchronizable content) for testing.
var tDM = &Entry{Contents: map[string]*Entry{
	"file":                          tF1,
	"unicode-composed-\xc3\xa9ntry": tF1,
	"second_file.txt":               tF2,
	"executable file":               tF3E,
	"file link":                     tSR,
	"subdir":                        tD0,
	"populated subdir":              tD1,
}}

// tDMSU is a version of tDM where the symbolic link has been converted to
// untracked content.
var tDMSU = &Entry{Contents: map[string]*Entry{
	"file":                          tF1,
	"unicode-composed-\xc3\xa9ntry": tF1,
	"second_file.txt":               tF2,
	"executable file":               tF3E,
	"file link":                     tU,
	"subdir":                        tD0,
	"populated subdir":              tD1,
}}

// tDMU is an extended version of tDM containing unsynchronizable content. Its
// contents are a superset of the contents of tDM.
var tDMU = &Entry{Contents: map[string]*Entry{
	"file":                          tF1,
	"unicode-composed-\xc3\xa9ntry": tF1,
	"second_file.txt":               tF2,
	"executable file":               tF3E,
	"file link":                     tSR,
	"subdir":                        tD0,
	"populated subdir":              tD1,
	"problematic":                   tP1,
	"untracked":                     tU,
}}

// tDU is a directory entry (containing tU with name "untracked") for testing.
var tDU = &Entry{Contents: map[string]*Entry{"untracked": tU}}

// tDP1 is a directory entry (containing tP1 with name "problematic") for
// testing.
var tDP1 = &Entry{Contents: map[string]*Entry{"problematic": tP1}}

// tDP2 is a directory entry (containing tP2 with name "problematic") for
// testing.
var tDP2 = &Entry{Contents: map[string]*Entry{"problematic": tP2}}

// tIDDE is an invalid directory entry (with an empty but non-nil file digest)
// for testing.
var tIDDE = &Entry{Digest: []byte{}}

// tIDD is an invalid directory entry (with a file digest) for testing.
var tIDD = &Entry{Digest: testingDigest("invalid contents")}

// tIDE is an invalid directory entry (with executability set) for testing.
var tIDE = &Entry{Executable: true}

// tIDT is an invalid directory entry (with a symbolic link target) for testing.
var tIDT = &Entry{Target: "invalid target"}

// tIDP is an invalid directory entry (with a problem) for testing.
var tIDP = &Entry{Problem: "invalid problem"}

// tIDCE is an invalid directory (containing a file with an empty name) for
// testing.
var tIDCE = &Entry{Contents: map[string]*Entry{"": tF1}}

// tIDCD is an invalid directory (containing a file with a dot as its name) for
// testing.
var tIDCD = &Entry{Contents: map[string]*Entry{".": tF1}}

// tIDCDD is an invalid directory (containing a file with a double dot as its
// name) for testing.
var tIDCDD = &Entry{Contents: map[string]*Entry{"..": tF1}}

// tIDCS is an invalid directory (containing a file with a forward slash (the
// Mutagen path separator) in its name) for testing.
var tIDCS = &Entry{Contents: map[string]*Entry{"invalid/name": tF1}}

// tIDCN is an invalid directory (containing a nil entry) for testing.
var tIDCN = &Entry{Contents: map[string]*Entry{"invalid": tN}}

// tIFCE is an invalid file entry (with an empty but non-nil content map) for
// testing.
var tIFCE = &Entry{Kind: EntryKind_File, Contents: map[string]*Entry{}}

// tIFC is an invalid file entry (with a non-empty content map) for testing.
var tIFC = &Entry{Kind: EntryKind_File, Contents: map[string]*Entry{"invalid": tF1}}

// tIFT is an invalid file entry (with a symbolic link target) for testing.
var tIFT = &Entry{Kind: EntryKind_File, Target: "invalid target"}

// tIFP is an invalid file entry (with a problem) for testing.
var tIFP = &Entry{Kind: EntryKind_File, Problem: "invalid problem"}

// tIFDN is an invalid file entry (with a nil digest) for testing.
var tIFDN = &Entry{Kind: EntryKind_File}

// tIFDE is an invalid file entry (with an empty digest) for testing.
var tIFDE = &Entry{Kind: EntryKind_File, Digest: []byte{}}

// tISCE is an invalid symbolic link entry (with an empty but non-nil content
// map) for testing.
var tISCE = &Entry{Kind: EntryKind_SymbolicLink, Contents: map[string]*Entry{}}

// tISC is an invalid symbolic link entry (with a non-empty content map) for
// testing.
var tISC = &Entry{Kind: EntryKind_SymbolicLink, Contents: map[string]*Entry{"invalid": tF1}}

// tISDE is an invalid symbolic link entry (with an empty but non-nil file
// digest) for testing.
var tISDE = &Entry{Kind: EntryKind_SymbolicLink, Digest: []byte{}}

// tISD is an invalid symbolic link entry (with a file digest) for testing.
var tISD = &Entry{Kind: EntryKind_SymbolicLink, Digest: testingDigest("invalid contents")}

// tISE is an invalid symbolic link entry (with executability set) for testing.
var tISE = &Entry{Kind: EntryKind_SymbolicLink, Executable: true}

// tISP is an invalid symbolic link entry (with a problem) for testing.
var tISP = &Entry{Kind: EntryKind_SymbolicLink, Problem: "invalid problem"}

// tISTE is an invalid symbolic link entry (with an empty target) for testing.
var tISTE = &Entry{Kind: EntryKind_SymbolicLink}

// tIUCE is an invalid untracked entry (with an empty but non-nil content
// map) for testing.
var tIUCE = &Entry{Kind: EntryKind_Untracked, Contents: map[string]*Entry{}}

// tIUC is an invalid untracked entry (with a non-empty content map) for
// testing.
var tIUC = &Entry{Kind: EntryKind_Untracked, Contents: map[string]*Entry{"invalid": tF1}}

// tIUDE is an invalid untracked entry (with an empty but non-nil file
// digest) for testing.
var tIUDE = &Entry{Kind: EntryKind_Untracked, Digest: []byte{}}

// tIUD is an invalid untracked entry (with a file digest) for testing.
var tIUD = &Entry{Kind: EntryKind_Untracked, Digest: testingDigest("invalid contents")}

// tIUE is an invalid untracked entry (with executability set) for testing.
var tIUE = &Entry{Kind: EntryKind_Untracked, Executable: true}

// tIUT is an invalid untracked entry (with a symbolic link target) for testing.
var tIUT = &Entry{Kind: EntryKind_Untracked, Target: "invalid target"}

// tIUP is an invalid untracked entry (with a problem) for testing.
var tIUP = &Entry{Kind: EntryKind_Untracked, Problem: "invalid problem"}

// tIPCE is an invalid problematic entry (with an empty but non-nil content map)
// for testing.
var tIPCE = &Entry{Kind: EntryKind_Problematic, Contents: map[string]*Entry{}}

// tIPC is an invalid problematic entry (with a non-empty content map) for
// testing.
var tIPC = &Entry{Kind: EntryKind_Problematic, Contents: map[string]*Entry{"invalid": tF1}}

// tIPDE is an invalid problematic entry (with an empty but non-nil file digest)
// for testing.
var tIPDE = &Entry{Kind: EntryKind_Problematic, Digest: []byte{}}

// tIPD is an invalid problematic entry (with a file digest) for testing.
var tIPD = &Entry{Kind: EntryKind_Problematic, Digest: testingDigest("invalid contents")}

// tIPE is an invalid problematic entry (with executability set) for testing.
var tIPE = &Entry{Kind: EntryKind_Problematic, Executable: true}

// tIPT is an invalid problematic entry (with a symbolic link target) for
// testing.
var tIPT = &Entry{Kind: EntryKind_Problematic, Target: "invalid target"}

// tIPPE is an invalid problematic entry (with an empty problem) for testing.
var tIPPE = &Entry{Kind: EntryKind_Problematic}

// tII is an invalid entry of unknown kind.
var tII = &Entry{Kind: EntryKind(-1)}

// tF1ContentMap is a content map for tF1.
var tF1ContentMap = testingContentMap{"": []byte(tF1Content)}

// tF2ContentMap is a content map for tF2.
var tF2ContentMap = testingContentMap{"": []byte(tF2Content)}

// tF3ContentMap is a content map for tF3 and tF3E.
var tF3ContentMap = testingContentMap{"": []byte(tF3Content)}

// tD1ContentMap is a content map for tD1 and tDSR.
var tD1ContentMap = testingContentMap{"file": []byte(tF1Content)}

// tD2ContentMap is a content map for tD2.
var tD2ContentMap = testingContentMap{"file": []byte(tF2Content)}

// tD3ContentMap is a content map for tD3 and tD3E.
var tD3ContentMap = testingContentMap{"file": []byte(tF3Content)}

// tDMContentMap is a content map for tDM and tDMU.
var tDMContentMap = testingContentMap{
	"file":                          []byte(tF1Content),
	"unicode-composed-\xc3\xa9ntry": []byte(tF1Content),
	"second_file.txt":               []byte(tF2Content),
	"executable file":               []byte(tF3Content),
	"populated subdir/file":         []byte(tF1Content),
}
