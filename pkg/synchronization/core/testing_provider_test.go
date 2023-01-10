package core

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"os"
	"path/filepath"
	"sync"
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
	// hasherLock serializes access to hasher.
	hasherLock sync.Mutex
	// hasher is the hasher to use when verifying content.
	hasher hash.Hash
}

// Provide implements the Provider interface for testProvider.
func (p *testingProvider) Provide(path string, digest []byte) (string, error) {
	// Grab the content for this path.
	content, ok := p.contentMap[path]
	if !ok {
		return filepath.Join(p.storage, "does_not_exist"), nil
	}

	// Grab the hasher lock and defer its release.
	p.hasherLock.Lock()
	defer p.hasherLock.Unlock()

	// Compute an address to store the file.
	p.hasher.Reset()
	p.hasher.Write(content)
	if !bytes.Equal(p.hasher.Sum(nil), digest) {
		return "", errors.New("requested entry digest does not match expected")
	}
	p.hasher.Write([]byte(path))
	address := p.hasher.Sum(nil)

	// Create a file to store the content.
	file, err := os.Create(filepath.Join(p.storage, hex.EncodeToString(address)))
	if err != nil {
		return "", fmt.Errorf("unable to create storage file: %w", err)
	}

	// Write content.
	_, err = file.Write(content)
	file.Close()
	if err != nil {
		os.Remove(file.Name())
		return "", fmt.Errorf("unable to write file contents: %w", err)
	}

	// Success.
	return file.Name(), nil
}
