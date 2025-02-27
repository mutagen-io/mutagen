package filesystem

import (
	"bytes"
	"os"
	"testing"

	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/must"
)

func TestMarkHidden(t *testing.T) {
	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})

	// Create a temporary file and defer its removal.
	hiddenFile, err := os.CreateTemp("", ".mutagen_filesystem_hidden")
	if err != nil {
		t.Fatal("unable to create temporary hiddenFile file:", err)
	}
	must.Close(hiddenFile, logger)
	defer must.OSRemove(hiddenFile.Name(), logger)

	// Ensure that we can mark it as hidden.
	if err := MarkHidden(hiddenFile.Name()); err != nil {
		t.Fatal("unable to mark file as hidden")
	}

	// TODO: Should we verify hidden attributes on Windows?
}
