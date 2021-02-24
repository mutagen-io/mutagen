package agent

import (
	"strings"
	"unicode/utf8"

	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/environment"
	"github.com/mutagen-io/mutagen/pkg/prompting"
)

// unameSToGOOS maps uname -s output values to their corresponding GOOS values.
// Although some Windows environments (Cygwin, MSYS, and MinGW) support uname,
// their values are handled by unameSIsWindowsPosix because they are so varied
// (their value depends on the POSIX environment and its version, the system
// architecture, and the NT kernel version).
var unameSToGOOS = map[string]string{
	"AIX":       "aix",
	"Darwin":    "darwin",
	"DragonFly": "dragonfly",
	"FreeBSD":   "freebsd",
	"Linux":     "linux",
	"NetBSD":    "netbsd",
	"OpenBSD":   "openbsd",
	"SunOS":     "solaris",
	// TODO: Add more obscure uname -s values as necessary, e.g.
	// debian/kFreeBSD, which returns "GNU/kFreeBSD".
}

// unameSIsWindowsPosix determines whether or not a uname -s output value
// represents a Windows POSIX environment.
func unameSIsWindowsPosix(value string) bool {
	return strings.HasPrefix(value, "CYGWIN") ||
		strings.HasPrefix(value, "MINGW") ||
		strings.HasPrefix(value, "MSYS")
}

// unameMToGOARCH maps uname -m output values to their corresponding GOARCH
// values.
var unameMToGOARCH = map[string]string{
	"i386":     "386",
	"i486":     "386",
	"i586":     "386",
	"i686":     "386",
	"x86_64":   "amd64",
	"amd64":    "amd64",
	"armv5l":   "arm",
	"armv6l":   "arm",
	"armv7l":   "arm",
	"armv8l":   "arm64",
	"aarch64":  "arm64",
	"arm64":    "arm64",
	"mips":     "mips",
	"mipsel":   "mipsle",
	"mips64":   "mips64",
	"mips64el": "mips64le",
	"ppc64":    "ppc64",
	"ppc64le":  "ppc64le",
	"riscv64":  "riscv64",
	"s390x":    "s390x",
	// TODO: Add any more obscure uname -m variations that we might encounter.
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
	"ARM":   "arm",
	"ARM64": "arm64",
	// TODO: Add IA64 (that's the key) if Go ever supports Itanium, though
	// they've pretty much stated that this will never happen:
	// https://groups.google.com/forum/#!topic/golang-nuts/RgGF1Dudym4
}

// probePOSIX performs platform probing over an agent transport, working under
// the assumption that the remote system is a POSIX system.
func probePOSIX(transport Transport) (string, string, error) {
	// Try to invoke uname and print kernel and machine name.
	unameSMBytes, err := output(transport, "uname -s -m")
	if err != nil {
		return "", "", errors.Wrap(err, "unable to invoke uname")
	} else if !utf8.Valid(unameSMBytes) {
		return "", "", errors.New("remote output is not UTF-8 encoded")
	}

	// Parse uname output.
	unameSM := strings.Split(strings.TrimSpace(string(unameSMBytes)), " ")
	if len(unameSM) != 2 {
		return "", "", errors.New("invalid uname output")
	}
	unameS := unameSM[0]
	unameM := unameSM[1]

	// Translate GOOS. Windows POSIX systems typically include their NT version
	// number in their uname -s output, so we have to handle those specially.
	var goos string
	var ok bool
	if unameSIsWindowsPosix(unameS) {
		goos = "windows"
	} else if goos, ok = unameSToGOOS[unameS]; !ok {
		return "", "", errors.New("unknown platform")
	}

	// Translate GOARCH. On AIX systems, uname -m returns the machine's serial
	// number, but modern AIX only runs on 64-bit PowerPC anyway (and that's
	// all we support), so we set the architecture directly.
	var goarch string
	if goos == "aix" {
		goarch = "ppc64"
	} else if goarch, ok = unameMToGOARCH[unameM]; !ok {
		return "", "", errors.New("unknown architecture")
	}

	// Success.
	return goos, goarch, nil
}

// probeWindows performs platform probing over an agent transport, working under
// the assumption that the remote system is a Windows system.
func probeWindows(transport Transport) (string, string, error) {
	// Attempt to dump the remote environment.
	outputBytes, err := output(transport, "cmd /c set")
	if err != nil {
		return "", "", errors.Wrap(err, "unable to invoke remote environment printing")
	} else if !utf8.Valid(outputBytes) {
		return "", "", errors.New("remote output is not UTF-8 encoded")
	}

	// Parse the output block into a series of KEY=value specifications.
	environment := environment.ParseBlock(string(outputBytes))

	// Extract the OS and PROCESSOR_ARCHITECTURE environment variables.
	var os, processorArchitecture string
	for _, e := range environment {
		if strings.HasPrefix(e, "OS=") {
			os = e[3:]
		} else if strings.HasPrefix(e, "PROCESSOR_ARCHITECTURE=") {
			processorArchitecture = e[23:]
		}
	}

	// Translate to GOOS.
	goos, ok := osEnvToGOOS[os]
	if !ok {
		return "", "", errors.New("unknown platform")
	}

	// Translate to GOARCH.
	goarch, ok := processorArchitectureEnvToGOARCH[processorArchitecture]
	if !ok {
		return "", "", errors.New("unknown architecture")
	}

	// Success.
	return goos, goarch, nil
}

// probe attempts to identify the properties of the target platform (namely
// GOOS, GOARCH, and whether or not it's a POSIX environment (which it might be
// even on Windows)) using the specified transport.
func probe(transport Transport, prompter string) (string, string, bool, error) {
	// Attempt to probe for a POSIX platform. This might apply to certain
	// Windows environments as well.
	if err := prompting.Message(prompter, "Probing endpoint (POSIX)..."); err != nil {
		return "", "", false, errors.Wrap(err, "unable to message prompter")
	}
	if goos, goarch, err := probePOSIX(transport); err == nil {
		return goos, goarch, true, nil
	}

	// If that fails, attempt a Windows fallback.
	if err := prompting.Message(prompter, "Probing endpoint (Windows)..."); err != nil {
		return "", "", false, errors.Wrap(err, "unable to message prompter")
	}
	if goos, goarch, err := probeWindows(transport); err == nil {
		return goos, goarch, false, nil
	}

	// Failure.
	return "", "", false, errors.New("exhausted probing methods")
}
