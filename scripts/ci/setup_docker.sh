#!/bin/bash

# Compute the executable extension for this platform.
EXE_EXT=""
if [[ "$TRAVIS_OS_NAME" == "windows" ]]; then
    EXE_EXT=".exe"
fi

# Build the HTTP demo server that will serve as the Dockerfile entry point. We
# have to disable cgo because to avoid creating dependencies on host libraries
# that might not exist inside the container.
CGO_ENABLED=0 go build \
    -o "scripts/ci/docker/${TRAVIS_OS_NAME}/httpdemo${EXE_EXT}" \
    github.com/mutagen-io/mutagen/pkg/integration/fixtures/httpdemo

# Print the Docker version.
docker version

# Build our image.
docker image build \
    --pull \
    --tag "${MUTAGEN_TEST_DOCKER_IMAGE_NAME}" \
    "scripts/ci/docker/${TRAVIS_OS_NAME}" || exit $?

# Remove the generated executable.
rm "scripts/ci/docker/${TRAVIS_OS_NAME}/httpdemo${EXE_EXT}"

# Start a container.
docker container run \
    --name "${MUTAGEN_TEST_DOCKER_CONTAINER_NAME}" \
    --detach \
    "${MUTAGEN_TEST_DOCKER_IMAGE_NAME}" || exit $?
