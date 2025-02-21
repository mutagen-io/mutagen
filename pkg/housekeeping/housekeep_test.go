package housekeeping

import (
	"bytes"
	"testing"

	"github.com/mutagen-io/mutagen/pkg/logging"
)

// TestHousekeep tests that Housekeep succeeds without panicking.
func TestHousekeep(_ *testing.T) {
	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})
	Housekeep(logger)
}

// TestHousekeepAgents tests that housekeepAgents succeeds without panicking.
func TestHousekeepAgents(_ *testing.T) {
	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})
	housekeepAgents(logger)
}

// TestHousekeepCaches tests that housekeepCaches succeeds without panicking.
func TestHousekeepCaches(_ *testing.T) {
	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})
	housekeepCaches(logger)
}

// TestHousekeepStaging tests that housekeepStaging succeeds without panicking.
func TestHousekeepStaging(_ *testing.T) {
	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})
	housekeepStaging(logger)
}
