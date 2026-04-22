# Testing

This document describes how Mutagen's tests are structured, how to run
them locally, and how CI exercises them.


## Running Tests Locally

### Basic test run

All package tests must be run sequentially due to shared daemon state
and IPC endpoints:

    go test -p 1 ./pkg/...

The `-p 1` flag is required. Parallel test execution will cause
failures due to daemon lock contention and IPC socket conflicts.

### Integration tests

Integration tests require a local agent bundle to be built first:

    go run scripts/build.go --mode=local
    MUTAGEN_TEST_END_TO_END=full go test -p 1 ./pkg/...

Without `MUTAGEN_TEST_END_TO_END`, integration tests are skipped.
The `full` value runs all integration tests; `slim` runs a reduced
set suitable for race detection.

### SSPL tests

Tests for SSPL-licensed code (fanotify, xxh128, zstd) are gated
behind a build tag:

    go test -v -tags mutagensspl ./sspl/...

### Race detection

Run with the race detector enabled using a slim integration test
set (the race detector significantly increases execution time):

    MUTAGEN_TEST_END_TO_END=slim go test -p 1 -race ./pkg/...

### Data directory

Mutagen tests use `~/.mutagen-dev` for development builds and
`~/.mutagen` for release builds. In sandboxed or restricted
environments, set `MUTAGEN_DATA_DIRECTORY` to a writable absolute
path before running tests.


## Environment Variables

The following variables control test behavior and scope:

| Variable | Values | Effect |
|----------|--------|--------|
| `MUTAGEN_TEST_END_TO_END` | `full`, `slim` | Enables end-to-end integration tests. `full` runs all integration tests; `slim` runs a reduced set. |
| `MUTAGEN_TEST_SSH` | `true` | Enables SSH transport tests. Requires a local SSH server accepting connections to `localhost`. |
| `MUTAGEN_TEST_DOCKER` | `true` | Enables Docker transport tests. Requires a running Docker daemon and a pre-built test container (see `scripts/ci/setup_docker.sh`). |
| `MUTAGEN_TEST_DOCKER_CONTAINER_NAME` | name | Specifies the Docker container name for transport tests (typically `mutagentester`). |
| `MUTAGEN_TEST_DOCKER_USERNAME` | name | Specifies the user inside the Docker test container (Windows only). |
| `MUTAGEN_TEST_ENABLE_SSPL` | `true` | Includes SSPL-licensed code in the main test run (adds `-tags mutagensspl`). |
| `MUTAGEN_TEST_FAT32_ROOT` | path | macOS/Windows: path to a FAT32 filesystem root for filesystem behavior tests. |
| `MUTAGEN_TEST_HFS_ROOT` | path | macOS: path to an HFS+ filesystem root. |
| `MUTAGEN_TEST_APFS_ROOT` | path | macOS: path to an APFS filesystem root. |
| `MUTAGEN_TEST_SUBFS_ROOT` | path | macOS: path to an HFS+ subfilesystem mounted under APFS. |
| `MUTAGEN_DATA_DIRECTORY` | path | Overrides the default Mutagen data directory for tests. |


## Platform-Specific Behavior

Tests vary by platform due to filesystem capabilities, transport
availability, and SSPL support:

| Aspect | macOS | Linux | Windows |
|--------|-------|-------|---------|
| **SSPL tests** | Disabled in main run | Enabled | Disabled in main run |
| **SSH transport** | Tested | Tested | Not tested |
| **Docker transport** | Not tested | Tested | Tested |
| **Race detection** | Full CI only | Always | Full CI only |
| **386-bit tests** | Not available | Always | Always |
| **Filesystem partitions** | FAT32, HFS+, APFS | None | FAT32 |

SSPL tests (`go test -tags mutagensspl ./sspl/...`) run separately
on all platforms regardless of the per-platform `MUTAGEN_TEST_ENABLE_SSPL`
setting.

See `scripts/ci/test_parameters.sh` for the exact per-platform
configuration.


## CI Pipeline

CI runs on GitHub Actions with two modes:

### Slim mode (pull requests)

Optimized for fast feedback on PR iterations:

- Analysis (`gofmt`, `go vet`, `go mod tidy` check) on all platforms.
- Full test suite with coverage profiling on all platforms.
- Race detection on Linux only (skipped on macOS and Windows).
- 386-bit tests on Linux and Windows.
- Slim builds (common platforms only) on all platforms.
- Docker transport tests on Windows skipped unless Docker-related
  files changed.
- Sidecar multi-platform builds skipped unless sidecar-related
  files changed.

### Full mode (merge queue, tags, master/release pushes, manual)

Complete validation before code lands:

- Everything in slim mode, plus:
- Race detection on all platforms.
- Docker transport tests on all platforms unconditionally.
- Full release cross-compilation on macOS (all target platforms,
  code signing when credentials are available).
- Sidecar multi-platform Docker image builds.
- SHA256 checksums and GPG signing (tag builds only).
- Notarization (tag builds only).
- Release artifact upload (tag builds only).

### CI environment variables

These variables are set by the CI workflow to control test behavior:

| Variable | Set When | Effect |
|----------|----------|--------|
| `MUTAGEN_CI_SKIP_RACE` | `true` | Slim CI on macOS/Windows. Skips the race detection test pass. |
| `MUTAGEN_CI_SKIP_DOCKER` | `true` | Slim CI on Windows when Docker files unchanged. Disables Docker transport tests. |
| `MUTAGEN_CI_FULL_BUILD` | `true` | Full CI on macOS. Switches from slim to release cross-compilation. |

### Change detection

A lightweight "Detect Changes" preamble job determines which areas
of the codebase changed in the PR:

- **docker**: Files matching `docker` or `pkg/integration` changed.
  Triggers Docker transport tests on Windows even in slim mode.
- **sidecar**: Files matching `sidecar`, `images/`, or `Dockerfile`
  changed. Triggers sidecar builds even in slim mode.

See `.github/workflows/ci.yml` for the full CI configuration.
