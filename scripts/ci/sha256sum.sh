#!/bin/bash

# Exit immediately on failure.
set -e

# Move to the release directory.
pushd build/release > /dev/null

# Compute SHA256 digests.
sha256sum mutagen_* > SHA256SUMS

# Sign the SHA256 digests file.
gpg --detach-sign --armor \
    --default-key "${SHA256_GPG_SIGNING_IDENTITY}" \
    --output SHA256SUMS.gpg \
    SHA256SUMS

# Leave the release directory.
popd > /dev/null
