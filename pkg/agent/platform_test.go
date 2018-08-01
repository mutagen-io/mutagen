package agent

import (
	"testing"
)

// TestUnameSIsWindowsPosix runs various test cases against unameSIsWindowsPosix
// to validate classification behavior.
func TestUnameSIsWindowsPosix(t *testing.T) {
	// Create test cases.
	testCases := map[string]bool{
		"CYGWIN_NT-6.1":  true,
		"MSYS_NT-6.1":    true,
		"MINGW32_NT-6.1": true,
		"Linux":          false,
	}

	// Run test cases.
	for u, e := range testCases {
		if unameSIsWindowsPosix(u) != e {
			t.Error("incorrect classification for", u)
		}
	}
}
