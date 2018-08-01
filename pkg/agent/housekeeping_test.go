package agent

import (
	"testing"
)

// TestHousekeeping tests that Housekeeping succeeds without panicing.
func TestHousekeeping(t *testing.T) {
	// Invoke housekeeping.
	Housekeep()
}
