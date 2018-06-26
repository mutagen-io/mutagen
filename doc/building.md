# Building

Mutagen is *not* `go get`able because it needs to cross-compile agent binaries
for remote platforms and generate a bundle of these binaries. In general, users
should download the release builds from the
[releases page](https://github.com/havoc-io/mutagen/releases/latest).

Mutagen can, however, be built locally for testing and development. Mutagen
needs to be checked out into your `$GOPATH` to build, which you can do with
`go get` (or a Git checkout):

    go get -d github.com/havoc-io/mutagen

Mutagen uses Git submodules for vendoring, so if you do a raw Git checkout (as
opposed to the `go get` command above), you'll want to run the following inside
the Mutagen source tree:

    git submodule update --init

To build a Mutagen release, use the build script:

    go run scripts/build.go --mode=release

macOS is currently the only platform that supports doing release builds, because
the macOS binaries require CGO support for file monitoring. Release builds will
be placed in a directory called `build` in the root of the Mutagen source tree,
though because this script uses `go get` internally, it will also generate
content in your `$GOPATH`.

Builds for development and testing are possible on other platforms. To build the
`mutagen` command and an agent binary for your local platform only, do:

    go run scripts/build.go

To build the `mutagen` command and an assortment of agent binaries for common
test platforms (including macOS, potentially without CGO support), do:

    go run scripts/build.go --mode=testing

Both of these latter options will place the `mutagen` binary in `$GOPATH/bin`,
so you can treat them almost like `go get`.


## Protocol Buffers code generation

Mutagen uses Protocol Buffers extensively, and as such needs to generate Go code
from `.proto` files. To avoid the need for developers (and CI systems) to have
the Protocol Buffers compiler installed, generated code is checked into the
repository. If a `.proto` file is modified, the generated code can be
regenerated for all of Mutagen using the code generation script:

    go run scripts/generate.go
