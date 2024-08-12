package main

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"github.com/klauspost/compress/gzip"

	"github.com/mutagen-io/mutagen/cmd"

	"github.com/mutagen-io/mutagen/pkg/agent"
	"github.com/mutagen-io/mutagen/pkg/mutagen"
)

const (
	// agentPackage is the Go package URL to use for building Mutagen agent
	// binaries.
	agentPackage = "github.com/mutagen-io/mutagen/cmd/mutagen-agent"
	// cliPackage is the Go package URL to use for building Mutagen binaries.
	cliPackage = "github.com/mutagen-io/mutagen/cmd/mutagen"

	// agentBuildSubdirectoryName is the name of the build subdirectory where
	// agent binaries are built.
	agentBuildSubdirectoryName = "agent"
	// cliBuildSubdirectoryName is the name of the build subdirectory where CLI
	// binaries are built.
	cliBuildSubdirectoryName = "cli"
	// releaseBuildSubdirectoryName is the name of the build subdirectory where
	// release bundles are built.
	releaseBuildSubdirectoryName = "release"

	// agentBaseName is the name of the Mutagen agent binary without any path or
	// extension.
	agentBaseName = "mutagen-agent"
	// cliBaseName is the name of the Mutagen binary without any path or
	// extension.
	cliBaseName = "mutagen"

	// minimumMacOSVersion is the minimum version of macOS that we'll support
	// (currently pinned to the oldest version of macOS that Mutagen's minimum
	// Go version supports).
	minimumMacOSVersion = "10.15"

	// minimumARMSupport is the value to pass to the GOARM environment variable
	// when building binaries. We currently specify support for ARMv5. This will
	// enable software-based floating point. For our use case, this is totally
	// fine, because we don't have any floating-point-heavy code, and the
	// resulting binary bloat is very minimal. This won't apply for arm64, which
	// always has hardware-based floating point support. For more information,
	// see: https://github.com/golang/go/wiki/GoArm
	minimumARMSupport = "5"
)

// Target specifies a GOOS/GOARCH combination.
type Target struct {
	// GOOS is the GOOS environment variable specification for the target.
	GOOS string
	// GOARCH is the GOARCH environment variable specification for the target.
	GOARCH string
}

// String generates a human-readable representation of the target.
func (t Target) String() string {
	return fmt.Sprintf("%s/%s", t.GOOS, t.GOARCH)
}

// Name generates a representation of the target that is suitable for paths and
// file names.
func (t Target) Name() string {
	return fmt.Sprintf("%s_%s", t.GOOS, t.GOARCH)
}

// ExecutableName formats executable names for the target.
func (t Target) ExecutableName(base string) string {
	// If we're on Windows, append a ".exe" extension.
	if t.GOOS == "windows" {
		return fmt.Sprintf("%s.exe", base)
	}

	// Otherwise return the base name unmodified.
	return base
}

// appendGoEnv modifies an environment specification to make the Go toolchain
// generate output for the target. It assumes that the resulting environment
// will be used with os/exec.Cmd and thus doesn't avoid duplicate variables.
func (t Target) appendGoEnv(environment []string) []string {
	// Override GOOS/GOARCH.
	environment = append(environment, fmt.Sprintf("GOOS=%s", t.GOOS))
	environment = append(environment, fmt.Sprintf("GOARCH=%s", t.GOARCH))

	// If we're building a macOS binary on macOS, then we enable cgo because
	// we'll need it to access the FSEvents API. We have to enable it explicitly
	// because Go won't enable it when cross compiling between different Darwin
	// architectures. We also need to tell the C compiler and external linker to
	// support older versions of macOS. These flags will tell the C compiler to
	// generate code compatible with the target version of macOS and tell the
	// external linker what value to embed for the LC_VERSION_MIN_MACOSX flag in
	// the resulting Mach-O binaries. Go's internal linker automatically
	// defaults to a relatively liberal (old) value for this flag, but since
	// we're using an external linker, it defaults to the current SDK version.
	//
	// For all other platforms, we disable cgo. This is essential for our Linux
	// CI setup, because we build agent executables during testing that we then
	// run inside Docker containers for our integration tests. These containers
	// typically run Alpine Linux, and if the agent binary is linked to C
	// libraries that only exist on the build system, then they won't work
	// inside the container. We can't disable cgo on a global basis though,
	// because it's needed for race condition testing. Another reason that it's
	// good to disable cgo when building agent binaries during testing is that
	// the release agent binaries will also have cgo disabled (except on macOS),
	// and we'll want to faithfully recreate that.
	if t.GOOS == "darwin" && runtime.GOOS == "darwin" {
		environment = append(environment, "CGO_ENABLED=1")
		environment = append(environment, fmt.Sprintf("CGO_CFLAGS=-mmacosx-version-min=%s", minimumMacOSVersion))
		environment = append(environment, fmt.Sprintf("CGO_LDFLAGS=-mmacosx-version-min=%s", minimumMacOSVersion))
	} else {
		environment = append(environment, "CGO_ENABLED=0")
	}

	// Set up ARM target support. See notes for definition of minimumARMSupport.
	// We don't need to unset any existing GOARM variables since they simply
	// won't be used if we're not targeting (non-64-bit) ARM systems.
	if t.GOARCH == "arm" {
		environment = append(environment, fmt.Sprintf("GOARM=%s", minimumARMSupport))
	}

	// Done.
	return environment
}

// IsCrossTarget determines whether or not the target represents a
// cross-compilation target (i.e. not the native target for the current Go
// toolchain).
func (t Target) IsCrossTarget() bool {
	return t.GOOS != runtime.GOOS || t.GOARCH != runtime.GOARCH
}

// IncludeAgentInSlimBuildModes indicates whether or not the target should have
// an agent binary included in the agent bundle in slim and release-slim modes.
func (t Target) IncludeAgentInSlimBuildModes() bool {
	return !t.IsCrossTarget() ||
		(t.GOOS == "darwin") ||
		(t.GOOS == "windows" && t.GOARCH == "amd64") ||
		(t.GOOS == "linux" && (t.GOARCH == "amd64" || t.GOARCH == "arm64")) ||
		(t.GOOS == "freebsd" && t.GOARCH == "amd64")
}

// BuildBundleInReleaseSlimMode indicates whether or not the target should have
// a release bundle built in release-slim mode.
func (t Target) BuildBundleInReleaseSlimMode() bool {
	return !t.IsCrossTarget() ||
		(t.GOOS == "darwin") ||
		(t.GOOS == "windows" && t.GOARCH == "amd64") ||
		(t.GOOS == "linux" && t.GOARCH == "amd64")
}

// Build executes a module-aware build of the specified package URL, storing the
// output of the build at the specified path.
func (t Target) Build(url, output string, enableSSPLEnhancements, disableDebug bool) error {
	// Compute the build command. If we don't need debugging, then we use the -s
	// and -w linker flags to omit the symbol table and debugging information.
	// This shaves off about 25% of the binary size and only disables debugging
	// (stack traces are still intact). For more information, see:
	// https://blog.filippo.io/shrink-your-go-binaries-with-this-one-weird-trick
	// In this case, we also trim the code paths stored in the executable, as
	// there's no use in having the full paths available.
	arguments := []string{"build", "-o", output}
	var tags []string
	if url == cliPackage {
		tags = append(tags, "mutagencli")
	}
	if url == agentPackage {
		tags = append(tags, "mutagenagent")
	}
	if enableSSPLEnhancements {
		tags = append(tags, "mutagensspl")
	}
	if len(tags) > 0 {
		arguments = append(arguments, "-tags", strings.Join(tags, ","))
	}
	if disableDebug {
		arguments = append(arguments, "-ldflags=-s -w", "-trimpath")
	}
	arguments = append(arguments, url)

	// Create the build command.
	builder := exec.Command("go", arguments...)

	// Set the environment.
	builder.Env = t.appendGoEnv(builder.Environ())

	// Forward input, output, and error streams.
	builder.Stdin = os.Stdin
	builder.Stdout = os.Stdout
	builder.Stderr = os.Stderr

	// Run the build.
	return builder.Run()
}

// targets encodes which combinations of GOOS and GOARCH we want to use for
// building agent and CLI binaries. We don't build every target at the moment,
// but we do list all potential targets here and comment out those we don't
// support. This list is created from https://golang.org/doc/install/source.
// Unfortunately there's no automated way to construct this list, but that's
// fine since we have to manually groom it anyway.
var targets = []Target{
	// Define AIX targets.
	{"aix", "ppc64"},

	// Define Android targets. We disable support for Android since it doesn't
	// have a clearly defined use case as a target platform, though there might
	// be certain development scenarios where it would make sense as an endpoint
	// (via a third-party SSH server on the device).
	// {"android", "386"},
	// {"android", "amd64"},
	// {"android", "arm"},
	// {"android", "arm64"},

	// Define macOS targets.
	{"darwin", "amd64"},
	{"darwin", "arm64"},

	// Define DragonFlyBSD targets.
	{"dragonfly", "amd64"},

	// Define FreeBSD targets.
	{"freebsd", "386"},
	{"freebsd", "amd64"},
	{"freebsd", "arm"},
	// TODO: The freebsd/arm64 port was added in Go 1.14, but for some reason
	// isn't documented at https://golang.org/doc/install/source. Submit a pull
	// request to add it to the Go documentation.
	{"freebsd", "arm64"},

	// Define illumos targets. We disable explicit support for illumos because
	// it's already effectively supported by our Solaris target. illumos is (at
	// least for Mutagen's purposes) an ABI-compatible superset of Solaris, so
	// there's no need for a separate build. Within the Go toolchain, runtime,
	// and standard library, most of illumos' support is provided by the Solaris
	// port. The "illumos" target even implies the "solaris" build constraint.
	// As such, the Solaris binaries should work fine for illumos distributions.
	// Also, the uname command on illumos returns the same kernel name ("SunOS")
	// as Solaris, so our probing wouldn't be able to identify illumos anyway.
	// {"illumos", "amd64"},

	// Define WebAssembly targets. We disable support for WebAssembly since it
	// doesn't make sense as a target platform.
	// {"js", "wasm"},

	// Define iOS/iPadOS/watchOS/tvOS targets. We disable support for these
	// since they don't make sense as target platforms.
	// TODO: The ios/amd64 port was added in Go 1.16, but for some reason isn't
	// documented at https://golang.org/doc/install/source. Submit a pull
	// request to add it to the Go documentation.
	// {"ios", "amd64"},
	// {"ios", "arm64"},

	// Define Linux targets.
	{"linux", "386"},
	{"linux", "amd64"},
	{"linux", "arm"},
	{"linux", "arm64"},
	// TODO: Assess whether or not we want to support LoongArch. Support was
	// added in Go 1.19, but it sounds like most real-world deployments use a
	// Linux kernel that's too old to support binaries compiled by the official
	// Go toolchain. If this situation changes, then it's certainly worth
	// enabling support. The code does build successfully on this architecture.
	// In this case, we'll also need to update platform detection with the
	// appropriate uname -m value.
	// {"linux", "loong64"},
	{"linux", "ppc64"},
	{"linux", "ppc64le"},
	{"linux", "mips"},
	{"linux", "mipsle"},
	{"linux", "mips64"},
	{"linux", "mips64le"},
	{"linux", "riscv64"},
	{"linux", "s390x"},

	// Define NetBSD targets.
	{"netbsd", "386"},
	{"netbsd", "amd64"},
	{"netbsd", "arm"},
	// TODO: The netbsd/arm64 port was added in Go 1.16, but for some reason
	// isn't documented at https://golang.org/doc/install/source. Submit a pull
	// request to add it to the Go documentation.
	{"netbsd", "arm64"},

	// Define OpenBSD targets.
	{"openbsd", "386"},
	{"openbsd", "amd64"},
	{"openbsd", "arm"},
	{"openbsd", "arm64"},
	// TODO: The openbsd/mips64 port was added in Go 1.16, but for some reason
	// isn't documented at https://golang.org/doc/install/source. Submit a pull
	// request to add it to the Go documentation.
	// TODO: The openbsd/mips64 port seems to be broken when using the Go sys
	// subrepository after v0.1.0 - the Go linker crashes with a segfault. The
	// port also doesn't seem to have been tested on the Go build dashboard for
	// quite some time, so its reliability at this point is suspect. Until the
	// picture there clarifies a bit, it's not worth letting this one port hold
	// back the others from receiving updates.
	// {"openbsd", "mips64"},

	// Define Plan 9 targets. We disable support for Plan 9 because it's missing
	// too many system calls and other APIs necessary for Mutagen to build. It
	// might make sense to support Plan 9 as an endpoint for certain development
	// scenarios, but it will take a significant amount of work just to get the
	// Mutagen agent to build.
	// {"plan9", "386"},
	// {"plan9", "amd64"},
	// {"plan9", "arm"},

	// Define Solaris targets.
	{"solaris", "amd64"},

	// Define Windows targets.
	{"windows", "386"},
	{"windows", "amd64"},
	// TODO: The windows/arm port was added in Go 1.12, but for some reason
	// isn't documented at https://golang.org/doc/install/source. Submit a pull
	// request to add it to the Go documentation.
	{"windows", "arm"},
	{"windows", "arm64"},
}

// macOSCodeSign performs macOS code signing on the specified path using the
// specified signing identity. It performs code signing in a manner suitable for
// later submission to Apple for notarization.
func macOSCodeSign(path, identity string) error {
	// Create the code signing command.
	//
	// We include the --force flag because the Go toolchain won't touch binaries
	// if they don't need to be rebuilt and thus we might have a signature from
	// a previous build. In that case, the code signing operation will fail
	// without --force. When --force is specified, any existing signature will
	// be overwritten, unless it's using the same code signing identity, in
	// which case it will simply be left in place (which is actually optimal for
	// for repeated local usage). Note that the --force flag is not required to
	// override ad hoc signatures (which the Go toolchain will add by default
	// darwin/arm64 binaries).
	//
	// The --options runtime and --timestamp flags are required to enable the
	// hardened runtime (which doesn't affect Mutagen binaries) and to use a
	// secure signing timestamp, both of which are required for notarization.
	codesign := exec.Command("codesign",
		"--sign", identity,
		"--force",
		"--options", "runtime",
		"--timestamp",
		"--verbose",
		path,
	)

	// Forward input, output, and error streams.
	codesign.Stdin = os.Stdin
	codesign.Stdout = os.Stdout
	codesign.Stderr = os.Stderr

	// Run code signing.
	return codesign.Run()
}

// archiveBuilderCopyBufferSize determines the size of the copy buffer used when
// generating archive files.
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
		return nil, fmt.Errorf("unable to create target file: %w", err)
	}

	// Create the compressor.
	compressor, err := gzip.NewWriterLevel(file, gzip.BestCompression)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("unable to create compressor: %w", err)
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
		return fmt.Errorf("unable to close archiver: %w", err)
	} else if err := b.compressor.Close(); err != nil {
		b.file.Close()
		return fmt.Errorf("unable to close compressor: %w", err)
	} else if err := b.file.Close(); err != nil {
		return fmt.Errorf("unable to close file: %w", err)
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
		return fmt.Errorf("unable to open file: %w", err)
	}
	defer file.Close()

	// Compute its size.
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("unable to determine file size: %w", err)
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
		return fmt.Errorf("unable to write archive header: %w", err)
	}

	// Copy the file contents.
	if _, err := io.CopyBuffer(b.archiver, file, b.copyBuffer); err != nil {
		return fmt.Errorf("unable to write archive entry: %w", err)
	}

	// Success.
	return nil
}

// copyFile copies the contents at sourcePath to a newly created file at
// destinationPath that inherits the permissions of sourcePath.
func copyFile(sourcePath, destinationPath string) error {
	// Open the source file and defer its closure.
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("unable to open source file: %w", err)
	}
	defer source.Close()

	// Grab source file metadata.
	metadata, err := source.Stat()
	if err != nil {
		return fmt.Errorf("unable to query source file metadata: %w", err)
	}

	// Remove the destination.
	os.Remove(destinationPath)

	// Create the destination file and defer its closure. We open with exclusive
	// creation flags to ensure that we're the ones creating the file so that
	// its permissions are set correctly.
	destination, err := os.OpenFile(destinationPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, metadata.Mode()&os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to create destination file: %w", err)
	}
	defer destination.Close()

	// Copy contents.
	if count, err := io.Copy(destination, source); err != nil {
		return fmt.Errorf("unable to copy data: %w", err)
	} else if count != metadata.Size() {
		return errors.New("copied size does not match expected")
	}

	// Success.
	return nil
}

var usage = `usage: build [-h|--help] [-m|--mode=<mode>] [--sspl]
       [--macos-codesign-identity=<identity>]

The mode flag accepts four values: 'local', 'slim', 'release', and
'release-slim'. 'local' will build CLI and agent binaries only for the current
platform. 'slim' will build the CLI binary for only the current platform and
agents for a common subset of platforms. 'release' will build CLI and agent
binaries for all platforms and package for release. 'release-slim' is the same
as release but only builds release bundles for a small subset of platforms. The
default mode is 'slim'.

If --sspl is specified, then SSPL-licensed enhancements will be included in the
build output. By default, only MIT-licensed code is included in builds.

If --macos-codesign-identity specifies a non-empty value, then it will be used
to perform code signing on all macOS binaries in a fashion suitable for
notarization by Apple. The codesign utility must be able to access the
associated certificate and private keys in Keychain Access without a password if
this script is operated in a non-interactive mode.
`

// build is the primary entry point.
func build() error {
	// Parse command line arguments.
	flagSet := pflag.NewFlagSet("build", pflag.ContinueOnError)
	flagSet.SetOutput(io.Discard)
	var mode, macosCodesignIdentity string
	var enableSSPLEnhancements bool
	flagSet.StringVarP(&mode, "mode", "m", "slim", "specify the build mode")
	flagSet.StringVar(&macosCodesignIdentity, "macos-codesign-identity", "", "specify the macOS code signing identity")
	flagSet.BoolVar(&enableSSPLEnhancements, "sspl", false, "enable SSPL-licensed enhancements")
	if err := flagSet.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			fmt.Fprint(os.Stdout, usage)
			return nil
		} else {
			return fmt.Errorf("unable to parse command line: %w", err)
		}
	}
	if !(mode == "local" || mode == "slim" || mode == "release" || mode == "release-slim") {
		return fmt.Errorf("invalid build mode: %s", mode)
	}

	// The only platform really suited to cross-compiling for every other
	// platform at the moment is macOS. This is because FSEvents is used for
	// file monitoring and that is a C-based API, not accessible purely via
	// system calls. All of the other platforms can operate with pure Go
	// compilation.
	if runtime.GOOS != "darwin" {
		if mode == "release" {
			return errors.New("macOS is required for release builds")
		} else if mode == "slim" || mode == "release-slim" {
			cmd.Warning("macOS agents will be built without cgo support")
		}
	}

	// If a macOS code signing identity has been specified, then make sure we're
	// in a mode where that makes sense.
	if macosCodesignIdentity != "" && runtime.GOOS != "darwin" {
		return errors.New("macOS is required for macOS code signing")
	}

	// Compute the path to the Mutagen source directory.
	mutagenSourcePath, err := mutagen.SourceTreePath()
	if err != nil {
		return fmt.Errorf("unable to compute Mutagen source tree path: %w", err)
	}

	// Verify that we're running inside the Mutagen source directory, otherwise
	// we can't rely on Go modules working.
	workingDirectory, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("unable to compute working directory: %w", err)
	}
	workingDirectoryRelativePath, err := filepath.Rel(mutagenSourcePath, workingDirectory)
	if err != nil {
		return fmt.Errorf("unable to determine working directory relative path: %w", err)
	}
	if strings.Contains(workingDirectoryRelativePath, "..") {
		return errors.New("build script run outside Mutagen source tree")
	}

	// Compute the path to the build directory and ensure that it exists.
	buildPath := filepath.Join(mutagenSourcePath, mutagen.BuildDirectoryName)
	if err := os.MkdirAll(buildPath, 0700); err != nil {
		return fmt.Errorf("unable to create build directory: %w", err)
	}

	// Create the necessary build directory hierarchy.
	agentBuildSubdirectoryPath := filepath.Join(buildPath, agentBuildSubdirectoryName)
	cliBuildSubdirectoryPath := filepath.Join(buildPath, cliBuildSubdirectoryName)
	releaseBuildSubdirectoryPath := filepath.Join(buildPath, releaseBuildSubdirectoryName)
	if err := os.MkdirAll(agentBuildSubdirectoryPath, 0700); err != nil {
		return fmt.Errorf("unable to create agent build subdirectory: %w", err)
	}
	if err := os.MkdirAll(cliBuildSubdirectoryPath, 0700); err != nil {
		return fmt.Errorf("unable to create CLI build subdirectory: %w", err)
	}
	if mode == "release" || mode == "release-slim" {
		if err := os.MkdirAll(releaseBuildSubdirectoryPath, 0700); err != nil {
			return fmt.Errorf("unable to create release build subdirectory: %w", err)
		}
	}

	// Compute the local target.
	localTarget := Target{runtime.GOOS, runtime.GOARCH}

	// Compute agent targets.
	var agentTargets []Target
	for _, target := range targets {
		if mode == "local" && target.IsCrossTarget() {
			continue
		} else if (mode == "slim" || mode == "release-slim") && !target.IncludeAgentInSlimBuildModes() {
			continue
		}
		agentTargets = append(agentTargets, target)
	}

	// Compute CLI targets.
	var cliTargets []Target
	for _, target := range targets {
		if (mode == "local" || mode == "slim") && target.IsCrossTarget() {
			continue
		} else if mode == "release-slim" && !target.BuildBundleInReleaseSlimMode() {
			continue
		}
		cliTargets = append(cliTargets, target)
	}

	// Determine whether or not to disable debugging information in binaries.
	// Doing so saves significant space, but is only suited to release builds.
	disableDebug := mode == "release" || mode == "release-slim"

	// Build agent binaries.
	log.Println("Building agent binaries...")
	for _, target := range agentTargets {
		log.Println("Building agent for", target)
		agentBuildPath := filepath.Join(agentBuildSubdirectoryPath, target.Name())
		if err := target.Build(agentPackage, agentBuildPath, enableSSPLEnhancements, disableDebug); err != nil {
			return fmt.Errorf("unable to build agent: %w", err)
		}
		if macosCodesignIdentity != "" && target.GOOS == "darwin" {
			if err := macOSCodeSign(agentBuildPath, macosCodesignIdentity); err != nil {
				return fmt.Errorf("unable to code sign agent for macOS: %w", err)
			}
		}
	}

	// Build CLI binaries.
	log.Println("Building CLI binaries...")
	for _, target := range cliTargets {
		log.Println("Building CLI for", target)
		cliBuildPath := filepath.Join(cliBuildSubdirectoryPath, target.Name())
		if err := target.Build(cliPackage, cliBuildPath, enableSSPLEnhancements, disableDebug); err != nil {
			return fmt.Errorf("unable to build CLI: %w", err)
		}
		if macosCodesignIdentity != "" && target.GOOS == "darwin" {
			if err := macOSCodeSign(cliBuildPath, macosCodesignIdentity); err != nil {
				return fmt.Errorf("unable to code sign CLI for macOS: %w", err)
			}
		}
	}

	// Build the agent bundle.
	log.Println("Building agent bundle...")
	agentBundlePath := filepath.Join(buildPath, agent.BundleName)
	agentBundleBuilder, err := NewArchiveBuilder(agentBundlePath)
	if err != nil {
		return fmt.Errorf("unable to create agent bundle archive builder: %w", err)
	}
	for _, target := range agentTargets {
		agentBuildPath := filepath.Join(agentBuildSubdirectoryPath, target.Name())
		if err := agentBundleBuilder.Add(target.Name(), agentBuildPath, 0755); err != nil {
			agentBundleBuilder.Close()
			return fmt.Errorf("unable to add agent to bundle: %w", err)
		}
	}
	if err := agentBundleBuilder.Close(); err != nil {
		return fmt.Errorf("unable to finalize agent bundle: %w", err)
	}

	// Build release bundles if necessary.
	if mode == "release" || mode == "release-slim" {
		log.Println("Building release bundles...")
		for _, target := range cliTargets {
			// Update status.
			log.Println("Building release bundle for", target)

			// Compute paths.
			cliBuildPath := filepath.Join(cliBuildSubdirectoryPath, target.Name())
			releaseBundlePath := filepath.Join(
				releaseBuildSubdirectoryPath,
				fmt.Sprintf("mutagen_%s_v%s.tar.gz", target.Name(), mutagen.Version),
			)

			// Build the release bundle.
			if releaseBundle, err := NewArchiveBuilder(releaseBundlePath); err != nil {
				return fmt.Errorf("unable to create release bundle: %w", err)
			} else if err = releaseBundle.Add(target.ExecutableName(cliBaseName), cliBuildPath, 0755); err != nil {
				releaseBundle.Close()
				return fmt.Errorf("unable to add CLI to release bundle: %w", err)
			} else if err = releaseBundle.Add("", agentBundlePath, 0644); err != nil {
				releaseBundle.Close()
				return fmt.Errorf("unable to add agent bundle to release bundle: %w", err)
			} else if err = releaseBundle.Close(); err != nil {
				return fmt.Errorf("unable to finalize release bundle: %w", err)
			}
		}
	}

	// Relocate the CLI binary for the current platform.
	log.Println("Copying binary for testing")
	localCLIBuildPath := filepath.Join(cliBuildSubdirectoryPath, localTarget.Name())
	localCLIRelocationPath := filepath.Join(buildPath, localTarget.ExecutableName(cliBaseName))
	if err := copyFile(localCLIBuildPath, localCLIRelocationPath); err != nil {
		return fmt.Errorf("unable to copy current platform CLI: %w", err)
	}

	// Success.
	return nil
}

func main() {
	if err := build(); err != nil {
		cmd.Fatal(err)
	}
}
