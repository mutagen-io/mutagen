# Building

Mutagen's build is slightly unique because it needs to cross-compile agent
binaries for remote platforms (with cgo support in the case of macOS) and then
generate a bundle of these binaries to ship alongside the Mutagen CLI. As such,
using `go get` or `go install` to acquire Mutagen will result in an incomplete
installation, and users should instead download the release builds from the
[releases page](https://github.com/mutagen-io/mutagen/releases/latest) or
[install Mutagen](https://mutagen.io/documentation/introduction/installation)
via [Homebrew](https://brew.sh/).

However, Mutagen can be built locally for testing and development. Mutagen
relies on the Go toolchain's module support, so make sure that you have Go
module support enabled.

Individual Mutagen executables can be built normally using the Go toolchain, but
a script is provided to ensure a normalized build, manage cross-compiled builds
and agent bundle creation, and perform code signing on macOS. To see information
about the build script, run:

    go run scripts/build.go --help

The build script can do four different types of builds: `local` (with support
for the local system only), `slim` (the default - with support for a selection
of common platforms used in testing), `release` (used for generating complete
release artifacts), and `release-slim` (used for generating complete release
artifacts for a selection of common platforms used in testing). macOS is
currently the only platform that supports doing `release` builds, because the
macOS binaries require cgo support for filesystem monitoring.

All artifacts from the build are placed in a `build` directory at the root of
the Mutagen source tree. As a convenience, artifacts built for the current
platform are placed in the root of the build directory for easy testing, e.g.:

    go run scripts/build.go
    build/mutagen --help


## Protocol Buffers code generation

Mutagen uses Protocol Buffers extensively, and as such needs to generate Go code
from `.proto` files. To avoid the need for developers (and CI systems) to have
the Protocol Buffers compiler installed, generated code is checked into the
repository. If a `.proto` file is modified, code can be regenerated by running

    go generate ./pkg/...

in the root of the Mutagen source tree.

The `go generate` commands used by Mutagen rely on Go module support being
enabled. You will also need to have the `protoc` compiler (with support for
Protocol Buffers 3) available in your path, but not the Go generator, which will
be built as part of the `go generate` command.
