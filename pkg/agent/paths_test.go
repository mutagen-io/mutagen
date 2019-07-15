package agent

import (
	"strings"
	"testing"

	"github.com/mutagen-io/mutagen/pkg/mutagen"
)

// TestInstallPath tests that the installPath method functions correctly. This
// has on-disk side-effects (namely creating the agents directory and the
// install directory for this version of Mutagen), but they should be harmless.
func TestInstallPath(t *testing.T) {
	// Verify that installPath succeeds.
	if p, err := installPath(); err != nil {
		t.Fatal("unable to compute/create install path:", err)
	} else if p == "" {
		t.Error("empty install path returned")
	} else if !strings.Contains(p, mutagen.Version) {
		t.Error("install path does not contain Mutagen version")
	}
}
