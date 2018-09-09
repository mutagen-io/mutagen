package filesystem

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"testing"
)

// TestTildeNotPathSeparator ensures that ~ is not considered a path separator
// on the platform. This is essentially guaranteed, but since we rely on this
// behavior, it's best to have an explicit check of it.
func TestTildeNotPathSeparator(t *testing.T) {
	if os.IsPathSeparator('~') {
		t.Fatal("tilde considered path separator")
	}
}

func TestTildeExpandHome(t *testing.T) {
	// Perform expansion.
	expanded, err := tildeExpand("~")
	if err != nil {
		t.Fatal("tilde expansion failed:", err)
	}

	// Ensure that the result matches the expected values.
	if expanded != HomeDirectory {
		t.Error("tilde-expanded path does not match expected")
	}
}

func TestTildeExpandHomeSlash(t *testing.T) {
	// Perform expansion.
	expanded, err := tildeExpand("~/")
	if err != nil {
		t.Fatal("tilde expansion failed:", err)
	}

	// Ensure that the result matches the expected values.
	if expanded != HomeDirectory {
		t.Error("tilde-expanded path does not match expected")
	}
}

func TestTildeExpandHomeBackslash(t *testing.T) {
	// Set expectations.
	expectFailure := runtime.GOOS != "windows"

	// Perform expansion.
	expanded, err := tildeExpand("~\\")
	if expectFailure && err == nil {
		t.Error("tilde expansion succeeded unexpectedly")
	} else if !expectFailure && err != nil {
		t.Fatal("tilde expansion failed:", err)
	}

	// Bail if we're done.
	if expectFailure {
		return
	}

	// Ensure that the result matches the expected values.
	if expanded != HomeDirectory {
		t.Error("tilde-expanded path does not match expected")
	}
}

func TestTildeExpandLookup(t *testing.T) {
	// Grab the current username.
	var username string
	if u, err := user.Current(); err != nil {
		t.Fatal("unable to look up current user:", err)
	} else {
		username = u.Username
	}

	// Perform expansion.
	expanded, err := tildeExpand("~" + username)
	if err != nil {
		t.Fatal("tilde expansion failed:", err)
	}

	// Ensure that the result matches the expected values.
	if expanded != HomeDirectory {
		t.Error("tilde-expanded path does not match expected")
	}
}

func TestTildeExpandLookupSlash(t *testing.T) {
	// Grab the current username.
	var username string
	if u, err := user.Current(); err != nil {
		t.Fatal("unable to look up current user:", err)
	} else {
		username = u.Username
	}

	// Perform expansion.
	expanded, err := tildeExpand(fmt.Sprintf("~%s/", username))
	if err != nil {
		t.Fatal("tilde expansion failed:", err)
	}

	// Ensure that the result matches the expected values.
	if expanded != HomeDirectory {
		t.Error("tilde-expanded path does not match expected")
	}
}

func TestTildeExpandLookupBackslash(t *testing.T) {
	// Set expectations.
	expectFailure := runtime.GOOS != "windows"

	// Grab the current username.
	var username string
	if u, err := user.Current(); err != nil {
		t.Fatal("unable to look up current user:", err)
	} else {
		username = u.Username
	}

	// Perform expansion.
	expanded, err := tildeExpand(fmt.Sprintf("~%s\\", username))
	if expectFailure && err == nil {
		t.Error("tilde expansion succeeded unexpectedly")
	} else if !expectFailure && err != nil {
		t.Fatal("tilde expansion failed:", err)
	}

	// Bail if we're done.
	if expectFailure {
		return
	}

	// Ensure that the result matches the expected values.
	if expanded != HomeDirectory {
		t.Error("tilde-expanded path does not match expected")
	}
}

func TestNormalizeHome(t *testing.T) {
	// Compute a path relative to the home directory.
	normalized, err := Normalize("~/somepath")
	if err != nil {
		t.Fatal("unable to normalize path:", err)
	}

	// Ensure that it's what we expect.
	if normalized != filepath.Join(HomeDirectory, "somepath") {
		t.Error("normalized path does not match expected")
	}
}
