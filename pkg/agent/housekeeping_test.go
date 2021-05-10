package agent

import (
	"testing"
)

// TestHousekeep tests that Housekeep succeeds without panicking.
func TestHousekeep(_ *testing.T) {
	Housekeep()
}
