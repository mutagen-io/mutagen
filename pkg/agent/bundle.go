package agent

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/process"
)

const (
	agentBundleName = "mutagen-agents.tar.gz"
)

var bundlePath string

func init() {
	// Compute the path to the agent bundle.
	bundlePath = filepath.Join(process.Current.ExecutableParentPath, agentBundleName)
}

// TODO: Note that this file will not mark the resultant file as executable.
func executableForPlatform(goos, goarch string) (string, error) {
	// Open the bundle path and ensure its closure.
	bundle, err := os.Open(bundlePath)
	if err != nil {
		return "", errors.Wrap(err, "unable to open agent bundle")
	}
	defer bundle.Close()

	// Create a decompressor and ensure its closure.
	bundleDecompressor, err := gzip.NewReader(bundle)
	if err != nil {
		return "", errors.Wrap(err, "unable to decompress agent bundle")
	}
	defer bundleDecompressor.Close()

	// Create an archive reader.
	bundleArchive := tar.NewReader(bundleDecompressor)

	// Scan until we find a matching header.
	var header *tar.Header
	for {
		if h, err := bundleArchive.Next(); err != nil {
			if err == io.EOF {
				break
			}
			return "", errors.Wrap(err, "unable to read archive header")
		} else if h.Name == fmt.Sprintf("%s_%s", goos, goarch) {
			header = h
			break
		}
	}

	// Check if we have a valid header. If not, there was no match.
	if header == nil {
		return "", errors.New("unsupported platform")
	}

	// Compute the base name for the output file.
	targetBaseName := process.ExecutableName(agentBaseName, goos)

	// Create a temporary file in which to receive the agent on disk.
	file, err := ioutil.TempFile("", targetBaseName)
	if err != nil {
		return "", errors.Wrap(err, "unable to create temporary file")
	}

	// Copy data into the file.
	if _, err := io.CopyN(file, bundleArchive, header.Size); err != nil {
		file.Close()
		os.Remove(file.Name())
		return "", errors.Wrap(err, "unable to copy agent data")
	}

	// Close the file.
	file.Close()

	// Success.
	return file.Name(), nil
}
