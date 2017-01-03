# Building

Mutagen is *not* `go get`able because it needs to cross-compile agent binaries
for remote platforms. In general, users should download the release builds from
the [releases page](https://github.com/havoc-io/mutagen/releases/latest).

Mutagen can, however, be built locally for testing and development. Mutagen
needs to be checked out into the `$GOROOT` to build, which you can do with
`go get` (or a Git checkout):

    go get -d github.com/havoc-io/mutagen

To build a Mutagen release, use the build script:

    go run scripts/build.go --mode=release

macOS is currently the only platform that supports doing release builds, because
the macOS binaries require CGO support for file monitoring. Release builds will
be placed in a directory called `build` in the root of the Mutagen source tree,
though because this script uses `go get` internally, it will also generate
content in your `$GOPATH`.

Builds for development and testing are possible on other platforms. To build for
your local platform only, do:

    go run scripts/build.go

To build for your local platform and an assortment of common test platforms
(including macOS without CGO support), do:

    go run scripts/build.go --mode=testing

Both of these latter options will place the `mutagen` binary in `$GOPATH/bin`,
so you can treat them almost like `go get`.
