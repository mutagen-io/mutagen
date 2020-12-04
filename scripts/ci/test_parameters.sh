#!/bin/bash

# Determine the operating system.
MUTAGEN_OS_NAME="$(go env GOOS)"

# Set common integration testing parameters.
export MUTAGEN_TEST_END_TO_END="full"

# Set platform-specific integration testing parameters.
if [[ "${MUTAGEN_OS_NAME}" == "darwin" ]]; then
    export MUTAGEN_TEST_SSH="true"
    export MUTAGEN_TEST_FAT32_ROOT="/Volumes/FAT32ROOT"
    export MUTAGEN_TEST_HFS_ROOT="/Volumes/HFSRoot"
    export MUTAGEN_TEST_APFS_ROOT="/Volumes/APFSRoot"
    export MUTAGEN_TEST_FAT32_SUBROOT="/Volumes/APFSRoot/FAT32SUB"
elif [[ "${MUTAGEN_OS_NAME}" == "linux" ]]; then
    export MUTAGEN_TEST_SSH="true"
    export MUTAGEN_TEST_DOCKER="true"
    export MUTAGEN_TEST_DOCKER_CONTAINER_NAME="mutagentester"
elif [[ "${MUTAGEN_OS_NAME}" == "windows" ]]; then
    export MUTAGEN_TEST_DOCKER="true"
    export MUTAGEN_TEST_DOCKER_CONTAINER_NAME="mutagentester"
    export MUTAGEN_TEST_DOCKER_USERNAME="george"
    export MUTAGEN_TEST_FAT32_ROOT='v:\'
else
    echo "Unknown or unsupported operating system" 1>&2
    exit 1
fi
