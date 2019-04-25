package sync

import (
	"testing"
)

func TestSymlinkModeUnmarshalIgnore(t *testing.T) {
	var mode SymlinkMode
	if err := mode.UnmarshalText([]byte("ignore")); err != nil {
		t.Fatal("unable to unmarshal text:", err)
	} else if mode != SymlinkMode_SymlinkModeIgnore {
		t.Error("unmarshalled mode does not match expected")
	}
}

func TestSymlinkModeUnmarshalPortable(t *testing.T) {
	var mode SymlinkMode
	if err := mode.UnmarshalText([]byte("portable")); err != nil {
		t.Fatal("unable to unmarshal text:", err)
	} else if mode != SymlinkMode_SymlinkModePortable {
		t.Error("unmarshalled mode does not match expected")
	}
}

func TestSymlinkModeUnmarshalPOSIXRaw(t *testing.T) {
	var mode SymlinkMode
	if err := mode.UnmarshalText([]byte("posix-raw")); err != nil {
		t.Fatal("unable to unmarshal text:", err)
	} else if mode != SymlinkMode_SymlinkModePOSIXRaw {
		t.Error("unmarshalled mode does not match expected")
	}
}

func TestSymlinkModeUnmarshalEmpty(t *testing.T) {
	var mode SymlinkMode
	if mode.UnmarshalText([]byte("")) == nil {
		t.Error("empty symlink mode successfully unmarshalled")
	}
}

func TestSymlinkModeUnmarshalInvalid(t *testing.T) {
	var mode SymlinkMode
	if mode.UnmarshalText([]byte("invalid")) == nil {
		t.Error("invalid symlink mode successfully unmarshalled")
	}
}

func TestSymlinkModeSupported(t *testing.T) {
	if SymlinkMode_SymlinkModeDefault.Supported() {
		t.Error("default symlink mode considered supported")
	}
	if !SymlinkMode_SymlinkModePortable.Supported() {
		t.Error("portable symlink mode considered unsupported")
	}
	if !SymlinkMode_SymlinkModeIgnore.Supported() {
		t.Error("ignore symlink mode considered unsupported")
	}
	if !SymlinkMode_SymlinkModePOSIXRaw.Supported() {
		t.Error("POSIX raw symlink mode considered unsupported")
	}
	if (SymlinkMode_SymlinkModePOSIXRaw + 1).Supported() {
		t.Error("invalid symlink mode considered supported")
	}
}

func TestSymlinkModeDescription(t *testing.T) {
	if description := SymlinkMode_SymlinkModeDefault.Description(); description != "Default" {
		t.Error("default symlink mode description incorrect:", description, "!=", "Default")
	}
	if description := SymlinkMode_SymlinkModePortable.Description(); description != "Portable" {
		t.Error("symlink mode portable description incorrect:", description, "!=", "Portable")
	}
	if description := SymlinkMode_SymlinkModeIgnore.Description(); description != "Ignore" {
		t.Error("symlink mode ignore description incorrect:", description, "!=", "Ignore")
	}
	if description := SymlinkMode_SymlinkModePOSIXRaw.Description(); description != "POSIX Raw" {
		t.Error("symlink mode POSIX raw description incorrect:", description, "!=", "POSIX Raw")
	}
	if description := (SymlinkMode_SymlinkModePOSIXRaw + 1).Description(); description != "Unknown" {
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
