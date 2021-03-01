package agent

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/mutagen"
	"github.com/mutagen-io/mutagen/pkg/process"
)

const (
	// BundleName is the base name of the agent bundle.
	BundleName = "mutagen-agents.tar.gz"
)

// BundleLocation encodes an expected location for the agent bundle.
type BundleLocation uint8

const (
	// BundleLocationDefault indicates that the ExecutableForPlatform function
	// should expect to find the agent bundle in the same directory as the
	// current executable or (if the current executable resides in the "bin"
	// directory of a Filesystem Hierarchy Standard layout) in the libexec
	// directory.
	BundleLocationDefault BundleLocation = iota
	// BundleLocationBuildDirectory indicates that the ExecutableForPlatform
	// function should expect to find the agent bundle in the Mutagen build
	// directory. This mode is only used during integration testing. It is
	// required because test executables will be built in temporary directories.
	BundleLocationBuildDirectory
)

// ExpectedBundleLocation specifies the expected agent bundle location. It is
// set by the time that init functions have completed. After that, it should
// only be set at the process entry point, before any code calls into the agent
// package.
var ExpectedBundleLocation BundleLocation

// ExecutableForPlatform attempts to locate the agent bundle and extract an
// agent executable for the specified target platform. If no output path is
// specified, then the extracted file will be in a temporary location accessible
// to only the user, will have an appropriate extension for the target platform,
// and will have the executability bit set if it makes sense. The path to the
// extracted file will be returned, and the caller is responsible for cleaning
// up the file if this function returns a nil error.
func ExecutableForPlatform(goos, goarch, outputPath string) (string, error) {
	// Compute the path to the location in which we expect to find the agent
	// bundle.
	var bundleSearchPaths []string
	if ExpectedBundleLocation == BundleLocationDefault {
		// Add the executable directory as a search path.
		if executablePath, err := os.Executable(); err != nil {
			return "", fmt.Errorf("unable to determine executable path: %w", err)
		} else {
			bundleSearchPaths = append(bundleSearchPaths, filepath.Dir(executablePath))
		}

		// If the executable is in what appears to be a Filesystem Hierarchy
		// Standard layout, then add the libexec directory as a search path.
		if libexecPath, err := filesystem.LibexecPath(); err == nil {
			bundleSearchPaths = append(bundleSearchPaths, libexecPath)
		}
	} else if ExpectedBundleLocation == BundleLocationBuildDirectory {
		if sourceTreePath, err := mutagen.SourceTreePath(); err != nil {
			return "", fmt.Errorf("unable to determine Mutagen source tree path: %w", err)
		} else {
			bundleSearchPaths = append(bundleSearchPaths, filepath.Join(sourceTreePath, mutagen.BuildDirectoryName))
		}
	} else {
		panic("invalid bundle location specification")
	}

	// Loop until we find a bundle file. If we fail to locate a bundle, then
	// abort. If we succeed, then defer its closure.
	var bundle *os.File
	for _, path := range bundleSearchPaths {
		bundlePath := filepath.Join(path, BundleName)
		if file, err := os.Open(bundlePath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", fmt.Errorf("unable to open agent bundle (%s): %w", bundlePath, err)
		} else if metadata, err := file.Stat(); err != nil {
			file.Close()
			return "", fmt.Errorf("unable to access agent bundle (%s) file metadata: %w", bundlePath, err)
		} else if metadata.Mode()&os.ModeType != 0 {
			file.Close()
			return "", fmt.Errorf("agent bundle (%s) is not a file", bundlePath)
		} else {
			bundle = file
			defer bundle.Close()
		}
	}
	if bundle == nil {
		return "", fmt.Errorf("unable to locate agent bundle (search paths: %v)", bundleSearchPaths)
	}

	// Create a decompressor and defer its closure.
	bundleDecompressor, err := gzip.NewReader(bundle)
	if err != nil {
		return "", fmt.Errorf("unable to decompress agent bundle: %w", err)
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
			return "", fmt.Errorf("unable to read archive header: %w", err)
		} else if h.Name == fmt.Sprintf("%s_%s", goos, goarch) {
			header = h
			break
		}
	}

	// Check if we have a valid header. If not, there was no match.
	if header == nil {
		return "", errors.New("unsupported platform")
	}

	// If an output path has been specified, then open the path for writing,
	// otherwise create a temporary file.
	var file *os.File
	if outputPath != "" {
		file, err = os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	} else {
		file, err = os.CreateTemp("", process.ExecutableName(BaseName+".*", goos))
	}
	if err != nil {
		return "", fmt.Errorf("unable to create output file: %w", err)
	}

	// Copy data into the file.
	if _, err := io.CopyN(file, bundleArchive, header.Size); err != nil {
		file.Close()
		os.Remove(file.Name())
		return "", fmt.Errorf("unable to copy agent data: %w", err)
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
			return "", fmt.Errorf("unable to make agent executable: %w", err)
		}
	}

	// Close the file.
	if err := file.Close(); err != nil {
		os.Remove(file.Name())
		return "", fmt.Errorf("unable to close temporary file: %w", err)
	}

	// Success.
	return file.Name(), nil
}
