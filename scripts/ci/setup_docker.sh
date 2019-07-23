#!/bin/bash

# Build the HTTP demo server that will serve as the Dockerfile entry point. We
# have to disable cgo because to avoid creating dependencies on host libraries
# that might not exist inside the container.
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
    -o scripts/ci/docker/linux/httpdemo \
    github.com/mutagen-io/mutagen/pkg/integration/fixtures/httpdemo

# Print the Docker version.
docker version

# Build our image.
docker image build \
    --pull \
    --tag "${MUTAGEN_TEST_DOCKER_IMAGE_NAME}" \
    scripts/ci/docker/linux || exit $?

# Remove the generated executable.
rm scripts/ci/docker/linux/httpdemo

# Start a container.
docker container run \
    --name "${MUTAGEN_TEST_DOCKER_CONTAINER_NAME}" \
    --detach \
    "${MUTAGEN_TEST_DOCKER_IMAGE_NAME}" || exit $?
