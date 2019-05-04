package local

import (
	"testing"
)

// TestHousekeepCaches tests that HousekeepCaches succeeds without panicking.
func TestHousekeepCaches(_ *testing.T) {
	HousekeepCaches()
}

// TestHousekeepStaging tests that HousekeepStaging succeeds without panicking.
func TestHousekeepStaging(_ *testing.T) {
	HousekeepStaging()
}
