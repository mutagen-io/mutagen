package agent

import (
	"strings"
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
