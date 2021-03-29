package core

import (
	"bytes"
	"errors"
	"fmt"
	"hash"
	"os"
)

// testingContentMap is an in-memory content map type used by testProvider.
type testingContentMap map[string][]byte

// testingProvider is an implementation of the Provider interfaces for tests. It
// loads file data from an in-memory content map.
type testingProvider struct {
	// storage is the temporary directory from which files should be served.
	storage string
	// contentMap is a map from path to file content.
	contentMap testingContentMap
	// hasher is the hasher to use when verifying content.
	hasher hash.Hash
}

// Provide implements the Provider interface for testProvider.
func (p *testingProvider) Provide(path string, digest []byte) (string, error) {
	// Grab the content for this path.
	content, ok := p.contentMap[path]
	if !ok {
		return "", os.ErrNotExist
	}

	// Ensure it matches the requested hash.
	p.hasher.Reset()
	p.hasher.Write(content)
	if !bytes.Equal(p.hasher.Sum(nil), digest) {
		return "", errors.New("requested entry digest does not match expected")
	}

	// Create a temporary file in the serving root.
	temporaryFile, err := os.CreateTemp(p.storage, "mutagen_provide")
	if err != nil {
		return "", fmt.Errorf("unable to create temporary file: %w", err)
	}

	// Write content.
	_, err = temporaryFile.Write(content)
	temporaryFile.Close()
	if err != nil {
		os.Remove(temporaryFile.Name())
		return "", fmt.Errorf("unable to write file contents: %w", err)
	}

	// Success.
	return temporaryFile.Name(), nil
}
