package filesystem

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestMarkHidden(t *testing.T) {
	// Create a temporary file and defer its removal.
	hiddenFile, err := ioutil.TempFile("", ".mutagen_filesystem_hidden")
	if err != nil {
		t.Fatal("unable to create temporary hiddenFile file:", err)
	}
	hiddenFile.Close()
	defer os.Remove(hiddenFile.Name())

	// Ensure that we can mark it as hidden.
	if err := MarkHidden(hiddenFile.Name()); err != nil {
		t.Fatal("unable to mark file as hidden")
	}

	// TODO: Should we verify hidden attributes on Windows?
}
