package agent

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/pkg/errors"

	uuid "github.com/satori/go.uuid"

	"github.com/havoc-io/mutagen/environment"
	"github.com/havoc-io/mutagen/process"
	"github.com/havoc-io/mutagen/url"
)

func InstallSelf() error {
	// Get the path to the agent executable.
	agentPath := process.Current.ExecutablePath

	// Compute the destination.
	destination, err := installPath()
	if err != nil {
		return errors.Wrap(err, "unable to compute agent destination")
	}

	// Relocate the agent.
	if err = os.Rename(agentPath, destination); err != nil {
		return errors.Wrap(err, "unable to relocate agent executable")
	}

	// Success.
	return nil
}

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

	// Create a temporary file in which to receive the agent on disk.
	file, err := ioutil.TempFile("", agentBaseName)
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

// unameSToGOOS maps uname -s output values to the corresponding GOOS value.
// Although some Windows environments (Cygwin, MSYS, and MinGW) support uname,
// their values are handled by unameSIsWindowsPosix because they are so varied
// (their value depends on the POSIX environment and its version, the system
// architecture, and the NT kernel version).
var unameSToGOOS = map[string]string{
	"Linux":     "linux",
	"Darwin":    "darwin",
	"FreeBSD":   "freebsd",
	"NetBSD":    "netbsd",
	"OpenBSD":   "openbsd",
	"DragonFly": "dragonfly",
	"SunOS":     "solaris",
	"Plan9":     "plan9",
}

func unameSIsWindowsPosix(unameS string) bool {
	return strings.HasPrefix(unameS, "CYGWIN") ||
		strings.HasPrefix(unameS, "MINGW") ||
		strings.HasPrefix(unameS, "MSYS")
}

var unameMToGOARCH = map[string]string{
	"i386":   "386",
	"i486":   "386",
	"i586":   "386",
	"i686":   "386",
	"x86_64": "amd64",
	"amd64":  "amd64",
	"armv5l": "arm",
	"armv6l": "arm",
	"armv7l": "arm",
	// TODO: Add armv8l (is that the uname -m for it?).
	// TODO: Add MIPS (need to figure out uname -m for each variation).
	// TODO: Add PowerPC (need to figure out uname -m for each variation).
}

// osEnvToGOOS maps the value of the "OS" environment variable on Windows to the
// corresponding GOOS. There's only one supported value, but we keep things this
// way for symmetry and extensibility.
var osEnvToGOOS = map[string]string{
	"Windows_NT": "windows",
}

// processorArchitectureEnvToGOARCH maps the value of the
// "PROCESSOR_ARCHITECTURE" environment variable on Windows to the corresponding
// GOARCH.
var processorArchitectureEnvToGOARCH = map[string]string{
	"x86":   "386",
	"AMD64": "amd64",
	// TODO: Add IA64 (that's the key) if Go ever supports Itanium, though
	// they've pretty much stated that this will never happen:
	// https://groups.google.com/forum/#!topic/golang-nuts/RgGF1Dudym4
}

func probeSSHPOSIX(prompter string, remote *url.SSHURL) (string, string, error) {
	// Try to invoke uname and print kernel and machine name.
	unameSMBytes, err := sshOutput(prompter, "Probing endpoint", remote, "uname -s -m")
	if err != nil {
		return "", "", errors.Wrap(err, "unable to invoke uname")
	}

	// Parse uname output.
	unameSM := strings.Split(strings.TrimSpace(string(unameSMBytes)), " ")
	if len(unameSM) != 2 {
		return "", "", errors.New("invalid uname output")
	}
	unameS := unameSM[0]
	unameM := unameSM[1]

	// Translate GOOS.
	var goos string
	if unameSIsWindowsPosix(unameS) {
		goos = "windows"
	} else if g, ok := unameSToGOOS[unameS]; ok {
		goos = g
	} else {
		return "", "", errors.New("unknown platform")
	}

	// Translate GOARCH.
	goarch, ok := unameMToGOARCH[unameM]
	if !ok {
		return "", "", errors.New("unknown architecture")
	}

	// Success.
	return goos, goarch, nil
}

func probeSSHWindows(prompter string, remote *url.SSHURL) (string, string, error) {
	// Try to print the remote environment.
	envBytes, err := sshOutput(prompter, "Probing endpoint", remote, "cmd /c set")
	if err != nil {
		return "", "", errors.Wrap(err, "unable to invoke set")
	}

	// Parse set output.
	env, err := environment.ParseBlock(string(envBytes))
	if err != nil {
		return "", "", errors.Wrap(err, "unable to parse environment")
	}

	// Translate GOOS.
	goos, ok := osEnvToGOOS[env["OS"]]
	if !ok {
		return "", "", errors.New("unknown platform")
	}

	// Translate GOARCH.
	goarch, ok := processorArchitectureEnvToGOARCH[env["PROCESSOR_ARCHITECTURE"]]
	if !ok {
		return "", "", errors.New("unknown architecture")
	}

	// Success.
	return goos, goarch, nil
}

// probeSSHPlatform attempts to identify the properties of the target platform,
// namely GOOS, GOARCH, and whether or not it's a POSIX environment (which it
// might be even on Windows).
func probeSSHPlatform(prompter string, remote *url.SSHURL) (string, string, bool, error) {
	// Attempt to probe for a POSIX platform. This might apply to certain
	// Windows environments as well.
	if goos, goarch, err := probeSSHPOSIX(prompter, remote); err == nil {
		return goos, goarch, true, nil
	}

	// If that fails, attempt a Windows fallback.
	if goos, goarch, err := probeSSHWindows(prompter, remote); err == nil {
		return goos, goarch, false, nil
	}

	// Failure.
	return "", "", false, errors.New("exhausted probing methods")
}

func installSSH(prompter string, remote *url.SSHURL) error {
	// Detect the target platform.
	goos, goarch, posix, err := probeSSHPlatform(prompter, remote)
	if err != nil {
		return errors.Wrap(err, "unable to probe remote platform")
	}

	// Find the appropriate agent binary. Ensure that it's cleaned up when we're
	// done with it.
	agent, err := executableForPlatform(goos, goarch)
	if err != nil {
		return errors.Wrap(err, "unable to get agent for platform")
	}
	defer os.Remove(agent)

	// Copy the agent to the remote. We use a unique identifier for the
	// temporary destination. For Windows remotes, we add a ".exe" suffix, which
	// will automatically make the file executable on the remote (POSIX systems
	// are handled separately below). For POSIX systems, we add a dot prefix to
	// hide the executable a bit.
	// HACK: This assumes that the SSH user's home directory is used as the
	// default destination directory for SCP copies. That should be true in
	// 99.9% of cases, but if it becomes a major issue, we'll need to use the
	// probe information to handle this more carefully.
	destination := agentBaseName + uuid.NewV4().String()
	if goos == "windows" {
		destination += ".exe"
	}
	if posix {
		destination = "." + destination
	}
	destinationURL := &url.SSHURL{
		Username: remote.Username,
		Hostname: remote.Hostname,
		Port:     remote.Port,
		Path:     destination,
	}
	if err := scp(prompter, "Copying agent", agent, destinationURL); err != nil {
		return errors.Wrap(err, "unable to copy agent binary")
	}

	// Invoke the remote installation. For POSIX remotes, we have to incorporate
	// a "chmod +x" in order for the remote to execute the installer. The POSIX
	// solution is necessary (as opposed to simply chmod'ing the file before
	// copy) because if an installer is sent from a Windows to a POSIX system
	// using SCP, there's no way to preserve the executability bit (since
	// Windows doesn't have one). This will also be applied to Windows POSIX
	// environments, but a "chmod +x" there will have no effect.
	// HACK: This assumes that the SSH user's home directory is used as the
	// default working directory for SSH commands. We have to do this because we
	// don't have a portable mechanism to invoke the command relative to the
	// user's home directory and we don't want to do a probe of the remote
	// system before invoking the endpoint. This assumption should be fine for
	// 99.9% of cases, but if it becomes a major issue, we'll need to use the
	// probe information to handle this more carefully.
	var installCommand string
	if posix {
		installCommand = fmt.Sprintf("chmod +x %s && ./%s --install", destination, destination)
	} else {
		installCommand = fmt.Sprintf("%s --install", destination)
	}
	if err := sshRun(prompter, "Installing agent", remote, installCommand); err != nil {
		return errors.Wrap(err, "unable to invoke installation")
	}

	// Success.
	return nil
}
