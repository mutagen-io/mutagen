package housekeeping

import (
	"testing"
)

// TestHousekeep tests that Housekeep succeeds without panicking.
func TestHousekeep(_ *testing.T) {
	Housekeep()
}

// TestHousekeepAgents tests that housekeepAgents succeeds without panicking.
func TestHousekeepAgents(_ *testing.T) {
	housekeepAgents()
}

// TestHousekeepCaches tests that housekeepCaches succeeds without panicking.
func TestHousekeepCaches(_ *testing.T) {
	housekeepCaches()
}

// TestHousekeepStaging tests that housekeepStaging succeeds without panicking.
func TestHousekeepStaging(_ *testing.T) {
	housekeepStaging()
}
