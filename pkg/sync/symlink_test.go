package sync

import (
	"testing"
)

func TestSymlinkModeSupportedForUsage(t *testing.T) {
	if SymlinkMode_SymlinkDefault.Supported() {
		t.Error("default symlink mode considered supported")
	}
	if !SymlinkMode_SymlinkPortable.Supported() {
		t.Error("portable symlink mode considered unsupported")
	}
	if !SymlinkMode_SymlinkIgnore.Supported() {
		t.Error("ignore symlink mode considered unsupported")
	}
	if !SymlinkMode_SymlinkPOSIXRaw.Supported() {
		t.Error("POSIX raw symlink mode considered unsupported")
	}
	if (SymlinkMode_SymlinkPOSIXRaw + 1).Supported() {
		t.Error("invalid symlink mode considered supported")
	}
}

func TestSymlinkModeDescription(t *testing.T) {
	if description := SymlinkMode_SymlinkDefault.Description(); description != "Default" {
		t.Error("symlink mode sane description incorrect:", description, "!=", "Default")
	}
	if description := SymlinkMode_SymlinkPortable.Description(); description != "Portable" {
		t.Error("symlink mode sane description incorrect:", description, "!=", "Portable")
	}
	if description := SymlinkMode_SymlinkIgnore.Description(); description != "Ignore" {
		t.Error("symlink mode ignore description incorrect:", description, "!=", "Ignore")
	}
	if description := SymlinkMode_SymlinkPOSIXRaw.Description(); description != "POSIX Raw" {
		t.Error("symlink mode POSIX raw description incorrect:", description, "!=", "POSIX Raw")
	}
	if description := (SymlinkMode_SymlinkPOSIXRaw + 1).Description(); description != "Unknown" {
		t.Error("invalid symlink mode description incorrect:", description, "!=", "Unknown")
	}
}

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
