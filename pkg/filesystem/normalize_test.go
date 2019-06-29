package filesystem

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pkg/errors"
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
	// Compute the path to the user's home directory.
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		t.Fatal("unable to compute home directory:", err)
	}

	// Perform expansion.
	expanded, err := tildeExpand("~")
	if err != nil {
		t.Fatal("tilde expansion failed:", err)
	}

	// Ensure that the result matches the expected values.
	if expanded != homeDirectory {
		t.Error("tilde-expanded path does not match expected")
	}
}

func TestTildeExpandHomeSlash(t *testing.T) {
	// Compute the path to the user's home directory.
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		t.Fatal("unable to compute home directory:", err)
	}

	// Perform expansion.
	expanded, err := tildeExpand("~/")
	if err != nil {
		t.Fatal("tilde expansion failed:", err)
	}

	// Ensure that the result matches the expected values.
	if expanded != homeDirectory {
		t.Error("tilde-expanded path does not match expected")
	}
}

func TestTildeExpandHomeBackslash(t *testing.T) {
	// Set expectations.
	expectFailure := runtime.GOOS != "windows"

	// Compute the path to the user's home directory.
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		t.Fatal("unable to compute home directory:", err)
	}

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
	if expanded != homeDirectory {
		t.Error("tilde-expanded path does not match expected")
	}
}

// currentUsername is a utility wrapper around user.Current for Windows systems,
// where the Username field will be of the form DOMAIN\username.
func currentUsername() (string, error) {
	// Grab the user.
	user, err := user.Current()
	if err != nil {
		return "", errors.Wrap(err, "unable to get current user")
	}

	// If we're on a POSIX system, we're done.
	if runtime.GOOS != "windows" {
		return user.Username, nil
	}

	// If we're on Windows, there may be a DOMAIN\ prefix on the username.
	if index := strings.IndexByte(user.Username, '\\'); index >= 0 {
		if index == len(user.Username) {
			return "", errors.New("domain extends to end of username")
		}
		return user.Username[index+1:], nil
	}
	return user.Username, nil
}

func TestTildeExpandLookup(t *testing.T) {
	// Compute the path to the user's home directory.
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		t.Fatal("unable to compute home directory:", err)
	}

	// Grab the current username.
	username, err := currentUsername()
	if err != nil {
		t.Fatal("unable to look up current username:", err)
	}

	// Perform expansion.
	expanded, err := tildeExpand("~" + username)
	if err != nil {
		t.Fatal("tilde expansion failed:", err)
	}

	// Ensure that the result matches the expected values.
	if expanded != homeDirectory {
		t.Error("tilde-expanded path does not match expected")
	}
}

func TestTildeExpandLookupSlash(t *testing.T) {
	// Compute the path to the user's home directory.
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		t.Fatal("unable to compute home directory:", err)
	}

	// Grab the current username.
	username, err := currentUsername()
	if err != nil {
		t.Fatal("unable to look up current username:", err)
	}

	// Perform expansion.
	expanded, err := tildeExpand(fmt.Sprintf("~%s/", username))
	if err != nil {
		t.Fatal("tilde expansion failed:", err)
	}

	// Ensure that the result matches the expected values.
	if expanded != homeDirectory {
		t.Error("tilde-expanded path does not match expected")
	}
}

func TestTildeExpandLookupBackslash(t *testing.T) {
	// Set expectations.
	expectFailure := runtime.GOOS != "windows"

	// Compute the path to the user's home directory.
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		t.Fatal("unable to compute home directory:", err)
	}

	// Grab the current username.
	username, err := currentUsername()
	if err != nil {
		t.Fatal("unable to look up current username:", err)
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
	if expanded != homeDirectory {
		t.Error("tilde-expanded path does not match expected")
	}
}

func TestNormalizeHome(t *testing.T) {
	// Compute the path to the user's home directory.
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		t.Fatal("unable to compute home directory:", err)
	}

	// Compute a path relative to the home directory.
	normalized, err := Normalize("~/somepath")
	if err != nil {
		t.Fatal("unable to normalize path:", err)
	}

	// Ensure that it's what we expect.
	if normalized != filepath.Join(homeDirectory, "somepath") {
		t.Error("normalized path does not match expected")
	}
}
