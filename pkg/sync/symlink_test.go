package sync

import (
	"testing"
)

func TestSymlinkModeSupported(t *testing.T) {
	if !SymlinkMode_Sane.Supported() {
		t.Error("symlink mode sane considered unsupported")
	}
	if !SymlinkMode_Ignore.Supported() {
		t.Error("symlink mode ignore considered unsupported")
	}
	if !SymlinkMode_POSIXRaw.Supported() {
		t.Error("symlink mode POSIX raw considered unsupported")
	}
	if (SymlinkMode_POSIXRaw + 1).Supported() {
		t.Error("invalid symlink mode considered supported")
	}
}

func TestSymlinkModeDescription(t *testing.T) {
	if description := SymlinkMode_Sane.Description(); description != "Sane" {
		t.Error("symlink mode sane description incorrect:", description, "!=", "Sane")
	}
	if description := SymlinkMode_Ignore.Description(); description != "Ignore" {
		t.Error("symlink mode ignore description incorrect:", description, "!=", "Ignore")
	}
	if description := SymlinkMode_POSIXRaw.Description(); description != "POSIX Raw" {
		t.Error("symlink mode POSIX raw description incorrect:", description, "!=", "POSIX Raw")
	}
	if description := (SymlinkMode_POSIXRaw + 1).Description(); description != "Unknown" {
		t.Error("invalid symlink mode description incorrect:", description, "!=", "Unknown")
	}
}

func TestSymlinkEmptyTargetInvalid(t *testing.T) {
	if _, err := normalizeSymlinkAndEnsureSane("file", ""); err == nil {
		t.Fatal("symlink with empty target treated as sane")
	}
}

const testLongSymlinkTarget = `dlaksjdflkajsfdlkajsdlfkjlkjlkajslfdkjlaksdjflkaj
dlaksjdflkajsfdlkajsdlfkjlkjlkajslfdkjlaksdjflkajasldkfakrjkjasdkfhajsfdhjasdhfa
dlaksjdflkajsfdlkajsdlfkjlkjlkajslfdkjlaksdjflkajasldkfakrjkjasdkfhajsfdhjasdhfa
dlaksjdflkajsfdlkajsdlfkjlkjlkajslfdkjlaksdjflkajasldkfakrjkjasdkfhajsfdhjasdhfa
dlaksjdflkajsfdlkajsdlfkjlkjlkajslfdkjlaksdjflkajasldkfakrjkjasdkfhajsfdhjasdhfa
dlaksjdflkajsfdlkajsdlfkjlkjlkajslfdkjlaksdjflkajasldkfakrjkjasdkfhajsfdhjasdhf`

func TestSymlinkTooLongInvalid(t *testing.T) {
	if _, err := normalizeSymlinkAndEnsureSane("file", testLongSymlinkTarget); err == nil {
		t.Fatal("symlink with overly long target treated as sane")
	}
}

func TestSymlinkWithColonInvalid(t *testing.T) {
	if _, err := normalizeSymlinkAndEnsureSane("file", "target:path"); err == nil {
		t.Fatal("symlink with colon in target treated as sane")
	}
}

func TestSymlinkAbsoluteInvalid(t *testing.T) {
	if _, err := normalizeSymlinkAndEnsureSane("file", "/target"); err == nil {
		t.Fatal("symlink with absolute target treated as sane")
	}
}

func TestSymlinkEscapesInvalid(t *testing.T) {
	if _, err := normalizeSymlinkAndEnsureSane("file", "../target"); err == nil {
		t.Fatal("symlink that escapes root treated as sane")
	}
}

func TestSymlinkEscapesDeeperInvalid(t *testing.T) {
	if _, err := normalizeSymlinkAndEnsureSane("directory/symlink path", "../../target"); err == nil {
		t.Fatal("symlink that escapes root treated as sane")
	}
}

func TestSymlinkSameDirectoryValid(t *testing.T) {
	if target, err := normalizeSymlinkAndEnsureSane("file", "other"); err != nil {
		t.Fatal("sane symlink treated as invalid:", err)
	} else if target != "other" {
		t.Error("sane symlink target incorrect:", target, "!=", "other")
	}
}

func TestSymlinkDotSameDirectoryValid(t *testing.T) {
	if target, err := normalizeSymlinkAndEnsureSane("file", "./other"); err != nil {
		t.Fatal("sane symlink treated as invalid:", err)
	} else if target != "./other" {
		t.Error("sane symlink target incorrect:", target, "!=", "./other")
	}
}

func TestSymlinkDotSubdirectoryValid(t *testing.T) {
	if target, err := normalizeSymlinkAndEnsureSane("file", "subdirectory/other"); err != nil {
		t.Fatal("sane symlink treated as invalid:", err)
	} else if target != "subdirectory/other" {
		t.Error("sane symlink target incorrect:", target, "!=", "subdirectory/other")
	}
}
