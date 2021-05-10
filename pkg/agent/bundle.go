package agent

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/mutagen"
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

// agentReadCloser is an io.ReadCloser implementation for reading agent
// executables from the agent bundle.
type agentReadCloser struct {
	// archive is the decompressed bundle archive stream, initially set to the
	// location of the executable within the bundle.
	archive io.Reader
	// decompressor is the decompressor.
	decompressor io.Closer
	// bundle is the bundle file.
	bundle io.Closer
}

// Read implements io.Reader.Read.
func (a *agentReadCloser) Read(buffer []byte) (int, error) {
	return a.archive.Read(buffer)
}

// Close implements io.Closer.Close.
func (a *agentReadCloser) Close() error {
	if err := a.decompressor.Close(); err != nil {
		a.bundle.Close()
		return err
	}
	return a.bundle.Close()
}

// executableStreamForPlatform attempts to locate the agent bundle and provides
// a stream that contains the agent executable for the specified platform. The
// caller is responsible for its closure (except in cases where a non-nil error
// is returned, in which case the stream will be nil).
func executableStreamForPlatform(goos, goarch string) (io.ReadCloser, error) {
	// Compute bundle search locations.
	var bundleSearchPaths []string
	if ExpectedBundleLocation == BundleLocationDefault {
		// Add the executable directory as a search path.
		if executablePath, err := os.Executable(); err != nil {
			return nil, fmt.Errorf("unable to determine executable path: %w", err)
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
			return nil, fmt.Errorf("unable to determine Mutagen source tree path: %w", err)
		} else {
			bundleSearchPaths = append(bundleSearchPaths, filepath.Join(sourceTreePath, mutagen.BuildDirectoryName))
		}
	} else {
		panic("invalid bundle location specification")
	}

	// Attempt to find the bundle file.
	var bundle *os.File
	for _, path := range bundleSearchPaths {
		bundlePath := filepath.Join(path, BundleName)
		if file, err := os.Open(bundlePath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("unable to open agent bundle (%s): %w", bundlePath, err)
		} else if metadata, err := file.Stat(); err != nil {
			file.Close()
			return nil, fmt.Errorf("unable to access agent bundle (%s) file metadata: %w", bundlePath, err)
		} else if metadata.Mode()&os.ModeType != 0 {
			file.Close()
			return nil, fmt.Errorf("agent bundle (%s) is not a file", bundlePath)
		} else {
			bundle = file
		}
	}
	if bundle == nil {
		return nil, fmt.Errorf("unable to locate agent bundle (search paths: %v): %w",
			bundleSearchPaths, os.ErrNotExist,
		)
	}

	// Create a decompressor and defer its closure.
	decompressor, err := gzip.NewReader(bundle)
	if err != nil {
		bundle.Close()
		return nil, fmt.Errorf("unable to decompress agent bundle: %w", err)
	}

	// Create an archive reader.
	archive := tar.NewReader(decompressor)

	// Scan until we find a matching header.
	var header *tar.Header
	target := fmt.Sprintf("%s_%s", goos, goarch)
	for {
		if h, err := archive.Next(); err != nil {
			if err == io.EOF {
				break
			}
			decompressor.Close()
			bundle.Close()
			return nil, fmt.Errorf("unable to read archive header: %w", err)
		} else if h.Name == target {
			header = h
			break
		}
	}

	// Check if we have a valid header. If not, then there was no match.
	if header == nil {
		decompressor.Close()
		bundle.Close()
		return nil, fmt.Errorf("unsupported platform: %w", os.ErrNotExist)
	}

	// Success.
	return &agentReadCloser{
		archive:      archive,
		decompressor: decompressor,
		bundle:       bundle,
	}, nil
}

// executableForPlatform is a wrapper around executableStreamForPlatform that
// writes the stream to a temporary file with secure permissions. The caller is
// responsible for cleaning up the file on disk.
func executableForPlatform(goos, goarch string) (string, error) {
	// Load the executable stream for the target platform and defer its closure.
	stream, err := executableStreamForPlatform(goos, goarch)
	if err != nil {
		return "", err
	}
	defer stream.Close()

	// Create the temporary file and defer its closure.
	file, err := os.CreateTemp("", BaseName)
	if err != nil {
		return "", nil
	}

	// Copy the executable contents.
	if _, err := io.Copy(file, stream); err != nil {
		file.Close()
		os.Remove(file.Name())
		return "", fmt.Errorf("unable to copy executable contents: %w", err)
	}

	// Close the file.
	if err := file.Close(); err != nil {
		os.Remove(file.Name())
		return "", fmt.Errorf("unable to close temporary file: %w", err)
	}

	// Success.
	return file.Name(), nil
}
