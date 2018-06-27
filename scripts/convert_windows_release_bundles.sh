#!/bin/bash

# Exit immediately on failure.
set -e

# Compute the version.
MUTAGEN_VERSION=$(mutagen version)

# Convert the 386 bundle.
tar xzf build/mutagen_windows_386_v${MUTAGEN_VERSION}.tar.gz
zip build/mutagen_windows_386_v${MUTAGEN_VERSION}.zip mutagen.exe mutagen-agents.tar.gz
rm mutagen.exe mutagen-agents.tar.gz

# Convert the amd64 bundle.
tar xzf build/mutagen_windows_amd64_v${MUTAGEN_VERSION}.tar.gz
zip build/mutagen_windows_amd64_v${MUTAGEN_VERSION}.zip mutagen.exe mutagen-agents.tar.gz
rm mutagen.exe mutagen-agents.tar.gz
