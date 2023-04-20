#!/bin/bash

# Exit immediately on failure.
set -e

# Determine the operating system.
MUTAGEN_OS_NAME="$(go env GOOS)"

# Perform a build that's appropriate for the platform.
if [[ "${MUTAGEN_OS_NAME}" == "darwin" ]]; then
    # Generate the zip archives needed for uploading to the notarization server.
    /usr/bin/ditto -c -k --keepParent build/cli/darwin_amd64 "${RUNNER_TEMP}/notarize_cli_darwin_amd64.zip"
    /usr/bin/ditto -c -k --keepParent build/cli/darwin_arm64 "${RUNNER_TEMP}/notarize_cli_darwin_arm64.zip"
    /usr/bin/ditto -c -k --keepParent build/agent/darwin_amd64 "${RUNNER_TEMP}/notarize_agent_darwin_amd64.zip"
    /usr/bin/ditto -c -k --keepParent build/agent/darwin_arm64 "${RUNNER_TEMP}/notarize_agent_darwin_arm64.zip"

    # Notarize each archive individually.
    find "${RUNNER_TEMP}" -name 'notarize_*.zip' -exec \
        xcrun notarytool submit \
        --wait \
        --apple-id "${MACOS_NOTARIZE_APPLE_ID}" \
        --password "${MACOS_NOTARIZE_APP_SPECIFIC_PASSWORD}" \
        --team-id "${MACOS_NOTARIZE_TEAM_ID}" \
        {} \;

    # Remove the archives.
    find "${RUNNER_TEMP}" -name 'notarize_*.zip' -exec rm -rf {} \;
else
    echo "This script is not supported on this platform" 1>&2
    exit 1
fi
