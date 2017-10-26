package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen"
	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/environment"
)

const (
	agentPackage   = "github.com/havoc-io/mutagen/cmd/mutagen-agent"
	cliPackage     = "github.com/havoc-io/mutagen/cmd/mutagen"
	agentBaseName  = "mutagen-agent"
	cliBaseName    = "mutagen"
	bundleBaseName = "mutagen-agents.tar.gz"
	// If we're compiling for arm, then specify support for ARMv5. This will
	// enable software-based floating point. For our use case, this is totally
	// fine, because we don't have any numeric code, and the resulting binary
	// bloat is very minimal. This won't apply for arm64, which always has
	// hardware-based floating point support. For more information, see:
	// https://github.com/golang/go/wiki/GoArm.
	minimumARMSupport = "5"
)

var GOPATH, GOBIN string

func init() {
	// Compute the GOPATH.
	if gopath, ok := environment.Current["GOPATH"]; !ok {
		panic("unable to determine GOPATH")
	} else {
		GOPATH = gopath
	}

	// Compute the GOPATH bin directory and ensure it exists.
	GOBIN = filepath.Join(GOPATH, "bin")
	if err := os.MkdirAll(GOBIN, 0700); err != nil {
		panic(errors.Wrap(err, "unable to ensure GOPATH bin directory exists"))
	}
}

type Target struct {
	GOOS   string
	GOARCH string
}

func (t Target) String() string {
	return fmt.Sprintf("%s/%s", t.GOOS, t.GOARCH)
}

func (t Target) Name() string {
	return fmt.Sprintf("%s_%s", t.GOOS, t.GOARCH)
}

func (t Target) ExecutableName(base string) string {
	if t.GOOS == "windows" {
		return fmt.Sprintf("%s.exe", base)
	}
	return base
}

func (t Target) goEnv() []string {
	// Duplicate the existing environment.
	result := environment.CopyCurrent()

	// Override GOOS/GOARCH.
	result["GOOS"] = t.GOOS
	result["GOARCH"] = t.GOARCH

	// Set up ARM target support. See notes for definition of minimumARMSupport.
	if t.GOOS == "arm" {
		result["GOARM"] = minimumARMSupport
	}

	// Reformat.
	return environment.Format(result)
}

func (t Target) Get(url string) error {
	// Execute the build. We use the "-s -w" linker flags to omit the symbol
	// table and debugging information. This shaves off about 25% of the binary
	// size and only disables debugging (stack traces are still intact). For
	// more information, see:
	// https://blog.filippo.io/shrink-your-go-binaries-with-this-one-weird-trick
	getter := exec.Command("go", "get", "-v", "-ldflags=-s -w", url)
	getter.Env = t.goEnv()
	getter.Stdin = os.Stdin
	getter.Stdout = os.Stdout
	getter.Stderr = os.Stderr
	if err := getter.Run(); err != nil {
		return errors.Wrap(err, "compilation failed")
	}

	// Success.
	return nil
}

func (t Target) Cross() bool {
	return t.GOOS != runtime.GOOS || t.GOARCH != runtime.GOARCH
}

func (t Target) ExecutableBuildPath() string {
	// Compute the path to the Go bin directory.
	result := filepath.Join(GOPATH, "bin")

	// If we're cross-compiling, then add the target subdirectory.
	if t.Cross() {
		result = filepath.Join(result, t.Name())
	}

	// Done.
	return result
}

// targets encodes which combinations of GOOS and GOARCH we want to use for
// building agent and CLI binaries. We don't build every target at the moment,
// but we do list all potential targets here and comment out those we don't
// support. This list is created from https://golang.org/doc/install/source.
// Unfortunately there's no automated way to construct this list, but that's
// fine since we have to manually groom it anyway.
var targets = []Target{
	// We completely disable Android because it doesn't provide a useful shell
	// or SSH server.
	// {"android", "arm"},
	// We completely disable darwin/386 because Go only supports macOS 10.7+,
	// which is always able to run amd64 binaries.
	// {"darwin", "386"},
	{"darwin", "amd64"},
	// We completely disble darwin/arm and darwin/arm64 because no ARM-based
	// Darwin platforms (iOS, watchOS, tvOS) provide a useful shell or SSH
	// server.
	// {"darwin", "arm"},
	// TODO: Figure out why darwin/arm64 doesn't compile in any case anyway.
	// We're seeing this issue: https://github.com/golang/go/issues/16445. The
	// "resolution" there makes sense, except that darwin/arm compiles fine.
	// According to https://golang.org/cmd/cgo/, CGO is automatically disabled
	// when cross-compiling. It's not clear if CGO is being disabled for
	// darwin/arm (and not for darwin/arm64) or if the environment is just
	// broken for darwin/arm64. In either case, there's some bug that needs to
	// be fixed. Either CGO needs to be disabled automatically in both cases for
	// consistency, or the cross-compilation environment needs to be fixed.
	// {"darwin", "arm64"},
	{"dragonfly", "amd64"},
	{"freebsd", "386"},
	{"freebsd", "amd64"},
	{"freebsd", "arm"},
	{"linux", "386"},
	{"linux", "amd64"},
	{"linux", "arm"},
	{"linux", "arm64"},
	{"linux", "ppc64"},
	{"linux", "ppc64le"},
	{"linux", "mips"},
	{"linux", "mipsle"},
	{"linux", "mips64"},
	{"linux", "mips64le"},
	// TODO: This combination is valid but not listed on the "Installing Go from
	// source" page. Perhaps we should open a pull request to change that?
	{"linux", "s390x"},
	{"netbsd", "386"},
	{"netbsd", "amd64"},
	{"netbsd", "arm"},
	{"openbsd", "386"},
	{"openbsd", "amd64"},
	{"openbsd", "arm"},
	// We completely disable Plan 9 because it is just missing too many
	// facilities for Mutagen to build easily, even just the agent component.
	// TODO: We might be able to get Plan 9 functioning as an agent, but it's
	// going to take some serious playing around with source file layouts and
	// build tags. To get started looking into this, look for the !plan9 build
	// tag and see where the gaps are. Most of the problems revolve around the
	// syscall package, but none of that is necessary for the agent, so it can
	// probably be built.
	// {"plan9", "386"},
	// {"plan9", "amd64"},
	{"solaris", "amd64"},
	{"windows", "386"},
	{"windows", "amd64"},
}

// TODO: Figure out if we should set this on a per-machine basis. This value is
// taken from Go's io.Copy method, which defaults to allocating a 32k buffer if
// none is provided.
const archiveBuilderCopyBufferSize = 32 * 1024

type ArchiveBuilder struct {
	file       *os.File
	compressor *gzip.Writer
	archiver   *tar.Writer
	copyBuffer []byte
}

func NewArchiveBuilder(bundlePath string) (*ArchiveBuilder, error) {
	// Open the underlying file.
	file, err := os.Create(bundlePath)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create target file")
	}

	// Create the compressor.
	compressor, err := gzip.NewWriterLevel(file, gzip.BestCompression)
	if err != nil {
		file.Close()
		return nil, errors.Wrap(err, "unable to create compressor")
	}

	// Success.
	return &ArchiveBuilder{
		file:       file,
		compressor: compressor,
		archiver:   tar.NewWriter(compressor),
		copyBuffer: make([]byte, archiveBuilderCopyBufferSize),
	}, nil
}

func (b *ArchiveBuilder) Close() error {
	// Close in the necessary order to trigger flushes.
	if err := b.archiver.Close(); err != nil {
		b.compressor.Close()
		b.file.Close()
		return errors.Wrap(err, "unable to close archiver")
	} else if err := b.compressor.Close(); err != nil {
		b.file.Close()
		return errors.Wrap(err, "unable to close compressor")
	} else if err := b.file.Close(); err != nil {
		return errors.Wrap(err, "unable to close file")
	}

	// Success.
	return nil
}

func (b *ArchiveBuilder) Add(name, path string, mode int64) error {
	// If the name is empty, use the base name.
	if name == "" {
		name = filepath.Base(path)
	}

	// Open the file and ensure its cleanup.
	file, err := os.Open(path)
	if err != nil {
		return errors.Wrap(err, "unable to open file")
	}
	defer file.Close()

	// Compute its size.
	stat, err := file.Stat()
	if err != nil {
		return errors.Wrap(err, "unable to determine file size")
	}
	size := stat.Size()

	// Write the header for the entry.
	header := &tar.Header{
		Name:    name,
		Mode:    mode,
		Size:    size,
		ModTime: time.Now(),
	}
	if err := b.archiver.WriteHeader(header); err != nil {
		return errors.Wrap(err, "unable to write archive header")
	}

	// Copy the file contents.
	if _, err := io.CopyBuffer(b.archiver, file, b.copyBuffer); err != nil {
		return errors.Wrap(err, "unable to write archive entry")
	}

	// Success.
	return nil
}

func buildAgentForTargetInTesting(target Target) bool {
	return !target.Cross() ||
		target.GOOS == "darwin" ||
		target.GOOS == "windows" ||
		(target.GOOS == "linux" &&
			(target.GOARCH == "amd64" ||
				target.GOARCH == "386" ||
				target.GOARCH == "arm"))
}

var buildUsage = `usage: build [-h|--help] [-m|--mode=<mode>]

The mode flag takes three values: 'slim', 'testing', and 'release'. 'slim' will
build binaries only for the current platform. 'testing' will build the CLI
binary for only the current platform and agents for a small subset of platforms.
Both 'slim' and 'testing' will place their build output (CLI binary and agent
bundle) in the '$GOPATH/bin' directory. 'release' will build CLI and agent
binaries for all platforms and package them in the 'build' subdirectory of the
Mutagen source tree. The default mode is 'slim'.
`

func main() {
	// Parse command line arguments.
	flagSet := cmd.NewFlagSet("build", buildUsage, nil)
	var mode string
	flagSet.StringVarP(&mode, "mode", "m", "slim", "specify the build mode")
	flagSet.ParseOrDie(os.Args[1:])
	if mode != "slim" && mode != "testing" && mode != "release" {
		cmd.Fatal(errors.New("invalid build mode"))
	}

	// The only platform really suited to cross-compiling for every other
	// platform at the moment is macOS. This is because its DNS resolution
	// really has to be done through the system's DNS resolver in order to
	// function properly and because FSEvents is used for file monitoring and
	// that is a C-based API, not accessible purely via system calls. All of the
	// other platforms can survive with pure Go compilation.
	if runtime.GOOS != "darwin" {
		if mode == "release" {
			cmd.Fatal(errors.New("macOS required for release builds"))
		} else if mode == "testing" {
			cmd.Warning("macOS agents will be built without cgo support")
		}
	}

	// Verify that this script is being run from the Mutagen source directory
	// located inside the user's GOPATH. This is just a sanity check to ensure
	// that people know what they're building.
	_, scriptPath, _, ok := runtime.Caller(0)
	if !ok {
		cmd.Fatal(errors.New("unable to compute script path"))
	}
	sourcePath, err := filepath.Abs(filepath.Dir(filepath.Dir(scriptPath)))
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to determine source path"))
	}
	expectedSourcePath, err := filepath.Abs(filepath.Join(
		GOPATH,
		"src",
		"github.com",
		"havoc-io",
		"mutagen",
	))
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to determine expected source path"))
	}
	if sourcePath != expectedSourcePath {
		cmd.Fatal(errors.New("script not invoked from target source path"))
	}

	// Compute the release build path and ensure it exists if we're in release
	// mode.
	var releasePath string
	if mode == "release" {
		releasePath = filepath.Join(sourcePath, "build")
		if err := os.MkdirAll(releasePath, 0700); err != nil {
			cmd.Fatal(errors.Wrap(err, "unable to create release directory"))
		}
	}

	// Create the agent bundle builder
	agentBundlePath := filepath.Join(GOBIN, bundleBaseName)
	agentBundle, err := NewArchiveBuilder(agentBundlePath)
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to create agent bundle builder"))
	}

	// Build and add agent binaries.
	for _, target := range targets {
		// Skip agent targets that aren't appropriate for this build mode.
		if mode == "slim" && target.Cross() {
			continue
		} else if mode == "testing" && !buildAgentForTargetInTesting(target) {
			continue
		}

		// Print information.
		fmt.Println("Building agent for", target)

		// Build.
		if err := target.Get(agentPackage); err != nil {
			cmd.Fatal(errors.Wrap(err, "unable to build agent"))
		}

		// Add to bundle.
		agentBuildPath := filepath.Join(
			target.ExecutableBuildPath(),
			target.ExecutableName(agentBaseName),
		)
		if err := agentBundle.Add(target.Name(), agentBuildPath, 0700); err != nil {
			cmd.Fatal(errors.Wrap(err, "unable to add agent to bundle"))
		}
	}

	// Close the agent bundle.
	if err := agentBundle.Close(); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to finalize agent bundle"))
	}

	// Build CLI binaries.
	for _, target := range targets {
		// If we're not in release mode, we don't do any cross-compilation.
		if mode != "release" && target.Cross() {
			continue
		}

		// Print information.
		fmt.Println("Building CLI components for", target)

		// Build.
		if err := target.Get(cliPackage); err != nil {
			cmd.Fatal(errors.Wrap(err, "unable to build CLI"))
		}

		// If we're in release mode, create the release bundle.
		if mode == "release" {
			// Print information.
			fmt.Println("Building release bundle for", target)

			// Compute CLI component paths.
			cliBuildPath := filepath.Join(
				target.ExecutableBuildPath(),
				target.ExecutableName(cliBaseName),
			)

			// Compute the bundle path.
			bundlePath := filepath.Join(
				releasePath,
				fmt.Sprintf("mutagen_%s_v%s.tar.gz", target.Name(), mutagen.Version),
			)

			// Create the bundle.
			bundle, err := NewArchiveBuilder(bundlePath)
			if err != nil {
				cmd.Fatal(errors.Wrap(err, "unable to create release bundle"))
			}

			// Add contents.
			if err := bundle.Add("", cliBuildPath, 0700); err != nil {
				cmd.Fatal(errors.Wrap(err, "unable to bundle CLI"))
			}
			if err := bundle.Add("", agentBundlePath, 0600); err != nil {
				cmd.Fatal(errors.Wrap(err, "unable to bundle agent bundle"))
			}

			// Close the bundle.
			if err := bundle.Close(); err != nil {
				cmd.Fatal(errors.Wrap(err, "unable to finalize release bundle"))
			}
		}
	}
}
