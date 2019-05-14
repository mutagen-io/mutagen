package agent

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/mutagen"
	"github.com/havoc-io/mutagen/pkg/process"
)

const (
	// BundleName is the base name of the agent bundle.
	BundleName = "mutagen-agents.tar.gz"
)

// BundleLocation encodes an expected location for the agent bundle.
type BundleLocation uint8

const (
	// BundleLocationSameDirectory specifies that the Mutagen executable in
	// which the agent package resides should expect to find the agent bundle in
	// the same directory as said executable.
	BundleLocationSameDirectory BundleLocation = iota
	// BundleLocationBuildDirectory specifies that the Mutagen executable in
	// which the agent package resides should expect to find the agent bundle in
	// the Mutagen build directory. This mode should only be used during
	// testing.
	BundleLocationBuildDirectory
)

// ExpectedBundleLocation specifies the expected agent bundle location. It is
// set by the time that init functions have completed. After that, it should
// only be set at the process entry point, before any code calls into the agent
// package.
var ExpectedBundleLocation BundleLocation

// executableForPlatform attempts to locate the agent bundle and extract an
// agent executable for the specified target platform. The extracted file will
// be in a temporary location accessible to only the user, and will have the
// executability bit set if it makes sense. The path to the extracted file will
// be returned, and the caller is responsible for cleaning up the file if this
// function returns a nil error.
func executableForPlatform(goos, goarch string) (string, error) {
	// Compute the path to the location in which we expect to find the agent
	// bundle.
	var bundleLocationPath string
	if ExpectedBundleLocation == BundleLocationSameDirectory {
		if executablePath, err := os.Executable(); err != nil {
			return "", errors.Wrap(err, "unable to determine executable path")
		} else {
			bundleLocationPath = filepath.Dir(executablePath)
		}
	} else if ExpectedBundleLocation == BundleLocationBuildDirectory {
		if sourceTreePath, err := mutagen.SourceTreePath(); err != nil {
			return "", errors.Wrap(err, "unable to determine Mutagen source tree path")
		} else {
			bundleLocationPath = filepath.Join(sourceTreePath, mutagen.BuildDirectoryName)
		}
	} else {
		panic("invalid bundle location specification")
	}

	// Compute the path to the agent bundle.
	bundlePath := filepath.Join(bundleLocationPath, BundleName)

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
	targetBaseName := process.ExecutableName(BaseName, goos)

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

	// If we're not on Windows and our target system is not Windows, mark the
	// file as executable. This will save us an additional "chmod +x" command
	// during agent installation. Note that the mechanism we use here
	// (os.File.Chmod) does not work on Windows (only the path-based os.Chmod is
	// supported there), but this is fine because this code wouldn't make sense
	// to use on Windows in any scenario (where executability bits don't exist).
	if runtime.GOOS != "windows" && goos != "windows" {
		if err := file.Chmod(0700); err != nil {
			file.Close()
			os.Remove(file.Name())
			return "", errors.Wrap(err, "unable to make agent executable")
		}
	}

	// Close the file.
	if err := file.Close(); err != nil {
		os.Remove(file.Name())
		return "", errors.Wrap(err, "unable to close temporary file")
	}

	// Success.
	return file.Name(), nil
}
