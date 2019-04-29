package agent

import (
	"strings"
	"unicode/utf8"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/prompt"
)

// unameSToGOOS maps uname -s output values to their corresponding GOOS values.
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
	"mips":     "mips",
	"mipsel":   "mipsle",
	"mips64":   "mips64",
	"mips64el": "mips64le",
	"ppc64":    "ppc64",
	"ppc64le":  "ppc64le",
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

	// Parse the output block into a series of VAR=value lines. First we replace
	// \r\n instances with \n, in case the block comes from Windows, trim any
	// outer whitespace (e.g. trailing newlines), and then split on newlines.
	// TODO: We might be able to switch this function to use a bufio.Scanner for
	// greater efficiency.
	output := string(outputBytes)
	output = strings.ReplaceAll(output, "\r\n", "\n")
	output = strings.TrimSpace(output)
	environment := strings.Split(output, "\n")

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
	if err := prompt.Message(prompter, "Probing endpoint (POSIX)..."); err != nil {
		return "", "", false, errors.Wrap(err, "unable to message prompter")
	}
	if goos, goarch, err := probePOSIX(transport); err == nil {
		return goos, goarch, true, nil
	}

	// If that fails, attempt a Windows fallback.
	if err := prompt.Message(prompter, "Probing endpoint (Windows)..."); err != nil {
		return "", "", false, errors.Wrap(err, "unable to message prompter")
	}
	if goos, goarch, err := probeWindows(transport); err == nil {
		return goos, goarch, false, nil
	}

	// Failure.
	return "", "", false, errors.New("exhausted probing methods")
}
