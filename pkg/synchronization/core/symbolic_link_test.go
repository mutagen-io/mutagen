package core

import (
	"testing"
)

func TestSymbolicLinkEmptyTargetInvalid(t *testing.T) {
	if _, err := normalizeSymbolicLinkAndEnsurePortable("file", ""); err == nil {
		t.Fatal("symbolic link with empty target treated as portable")
	}
}

const testLongSymbolicLinkTarget = `dlaksjdflkajsfdlkajsdlfkjlkjlkajslfdkjlaksdj
dlaksjdflkajsfdlkajsdlfkjlkjlkajslfdkjlaksdjflkajasldkfakrjkjasdkfhajsfdhjasdhfa
dlaksjdflkajsfdlkajsdlfkjlkjlkajslfdkjlaksdjflkajasldkfakrjkjasdkfhajsfdhjasdhfa
dlaksjdflkajsfdlkajsdlfkjlkjlkajslfdkjlaksdjflkajasldkfakrjkjasdkfhajsfdhjasdhfa
dlaksjdflkajsfdlkajsdlfkjlkjlkajslfdkjlaksdjflkajasldkfakrjkjasdkfhajsfdhjasdhfa
dlaksjdflkajsfdlkajsdlfkjlkjlkajslfdkjlaksdjflkajasldkfakrjkjasdkfhajsfdhjasdhf`

func TestSymbolicLinkTooLongInvalid(t *testing.T) {
	if _, err := normalizeSymbolicLinkAndEnsurePortable("file", testLongSymbolicLinkTarget); err == nil {
		t.Fatal("symbolic link with overly long target treated as portable")
	}
}

func TestSymbolicLinkWithColonInvalid(t *testing.T) {
	if _, err := normalizeSymbolicLinkAndEnsurePortable("file", "target:path"); err == nil {
		t.Fatal("symbolic link with colon in target treated as portable")
	}
}

func TestSymbolicLinkAbsoluteInvalid(t *testing.T) {
	if _, err := normalizeSymbolicLinkAndEnsurePortable("file", "/target"); err == nil {
		t.Fatal("symbolic link with absolute target treated as portable")
	}
}

func TestSymbolicLinkEscapesInvalid(t *testing.T) {
	if _, err := normalizeSymbolicLinkAndEnsurePortable("file", "../target"); err == nil {
		t.Fatal("symbolic link that escapes root treated as portable")
	}
}

func TestSymbolicLinkEscapesDeeperInvalid(t *testing.T) {
	if _, err := normalizeSymbolicLinkAndEnsurePortable("directory/symlink path", "../../target"); err == nil {
		t.Fatal("symbolic link that escapes root treated as portable")
	}
}

func TestSymbolicLinkSameDirectoryValid(t *testing.T) {
	if target, err := normalizeSymbolicLinkAndEnsurePortable("file", "other"); err != nil {
		t.Fatal("portable symbolic link treated as invalid:", err)
	} else if target != "other" {
		t.Error("normalized symbolic link target incorrect:", target, "!=", "other")
	}
}

func TestSymbolicLinkDotSameDirectoryValid(t *testing.T) {
	if target, err := normalizeSymbolicLinkAndEnsurePortable("file", "./other"); err != nil {
		t.Fatal("portable symbolic link treated as invalid:", err)
	} else if target != "./other" {
		t.Error("normalized symbolic link target incorrect:", target, "!=", "./other")
	}
}

func TestSymbolicLinkDotSubdirectoryValid(t *testing.T) {
	if target, err := normalizeSymbolicLinkAndEnsurePortable("file", "subdirectory/other"); err != nil {
		t.Fatal("portable symbolic link treated as invalid:", err)
	} else if target != "subdirectory/other" {
		t.Error("normalized symbolic link target incorrect:", target, "!=", "subdirectory/other")
	}
}
