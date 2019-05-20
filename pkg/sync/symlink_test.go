package sync

import (
	"testing"
)

func TestSymlinkEmptyTargetInvalid(t *testing.T) {
	if _, err := normalizeSymlinkAndEnsurePortable("file", ""); err == nil {
		t.Fatal("symlink with empty target treated as portable")
	}
}

const testLongSymlinkTarget = `dlaksjdflkajsfdlkajsdlfkjlkjlkajslfdkjlaksdjflkaj
dlaksjdflkajsfdlkajsdlfkjlkjlkajslfdkjlaksdjflkajasldkfakrjkjasdkfhajsfdhjasdhfa
dlaksjdflkajsfdlkajsdlfkjlkjlkajslfdkjlaksdjflkajasldkfakrjkjasdkfhajsfdhjasdhfa
dlaksjdflkajsfdlkajsdlfkjlkjlkajslfdkjlaksdjflkajasldkfakrjkjasdkfhajsfdhjasdhfa
dlaksjdflkajsfdlkajsdlfkjlkjlkajslfdkjlaksdjflkajasldkfakrjkjasdkfhajsfdhjasdhfa
dlaksjdflkajsfdlkajsdlfkjlkjlkajslfdkjlaksdjflkajasldkfakrjkjasdkfhajsfdhjasdhf`

func TestSymlinkTooLongInvalid(t *testing.T) {
	if _, err := normalizeSymlinkAndEnsurePortable("file", testLongSymlinkTarget); err == nil {
		t.Fatal("symlink with overly long target treated as portable")
	}
}

func TestSymlinkWithColonInvalid(t *testing.T) {
	if _, err := normalizeSymlinkAndEnsurePortable("file", "target:path"); err == nil {
		t.Fatal("symlink with colon in target treated as portable")
	}
}

func TestSymlinkAbsoluteInvalid(t *testing.T) {
	if _, err := normalizeSymlinkAndEnsurePortable("file", "/target"); err == nil {
		t.Fatal("symlink with absolute target treated as portable")
	}
}

func TestSymlinkEscapesInvalid(t *testing.T) {
	if _, err := normalizeSymlinkAndEnsurePortable("file", "../target"); err == nil {
		t.Fatal("symlink that escapes root treated as portable")
	}
}

func TestSymlinkEscapesDeeperInvalid(t *testing.T) {
	if _, err := normalizeSymlinkAndEnsurePortable("directory/symlink path", "../../target"); err == nil {
		t.Fatal("symlink that escapes root treated as portable")
	}
}

func TestSymlinkSameDirectoryValid(t *testing.T) {
	if target, err := normalizeSymlinkAndEnsurePortable("file", "other"); err != nil {
		t.Fatal("portable symlink treated as invalid:", err)
	} else if target != "other" {
		t.Error("normalized symlink target incorrect:", target, "!=", "other")
	}
}

func TestSymlinkDotSameDirectoryValid(t *testing.T) {
	if target, err := normalizeSymlinkAndEnsurePortable("file", "./other"); err != nil {
		t.Fatal("portable symlink treated as invalid:", err)
	} else if target != "./other" {
		t.Error("normalized symlink target incorrect:", target, "!=", "./other")
	}
}

func TestSymlinkDotSubdirectoryValid(t *testing.T) {
	if target, err := normalizeSymlinkAndEnsurePortable("file", "subdirectory/other"); err != nil {
		t.Fatal("portable symlink treated as invalid:", err)
	} else if target != "subdirectory/other" {
		t.Error("normalized symlink target incorrect:", target, "!=", "subdirectory/other")
	}
}
