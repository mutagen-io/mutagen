#!/bin/bash

# Exit immediately on failure.
set -e

# Determine the operating system.
MUTAGEN_OS_NAME="$(go env GOOS)"

# Perform a build that's appropriate for the platform.
if [[ "${MUTAGEN_OS_NAME}" == "darwin" ]]; then
    # Perform a full release build.
    go run scripts/build.go --mode=release

    # Determine the Mutagen version.
    MUTAGEN_VERSION="$(build/mutagen version)"

    # Convert the 386 bundle to zip format.
    tar xzf "build/release/mutagen_windows_386_v${MUTAGEN_VERSION}.tar.gz"
    zip "build/release/mutagen_windows_386_v${MUTAGEN_VERSION}.zip" mutagen.exe mutagen-agents.tar.gz
    rm mutagen.exe mutagen-agents.tar.gz

    # Convert the amd64 bundle to zip format.
    tar xzf "build/release/mutagen_windows_amd64_v${MUTAGEN_VERSION}.tar.gz"
    zip "build/release/mutagen_windows_amd64_v${MUTAGEN_VERSION}.zip" mutagen.exe mutagen-agents.tar.gz
    rm mutagen.exe mutagen-agents.tar.gz

    # Convert the arm bundle to zip format.
    tar xzf "build/release/mutagen_windows_arm_v${MUTAGEN_VERSION}.tar.gz"
    zip "build/release/mutagen_windows_arm_v${MUTAGEN_VERSION}.zip" mutagen.exe mutagen-agents.tar.gz
    rm mutagen.exe mutagen-agents.tar.gz
else
    go run scripts/build.go --mode=slim
fi

# Build test scripts to ensure that they are maintained as core packages evolve.
go build ./scripts/scan_bench
go build ./scripts/watch_demo
