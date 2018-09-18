#!/bin/bash

# Pull our base Docker image.
docker pull "${MUTAGEN_TEST_DOCKER_BASE_IMAGE_NAME}" || exit $?

# Build our image.
docker build \
    --tag "${MUTAGEN_TEST_DOCKER_IMAGE_NAME}" \
    --file dockerfile_linux \
    scripts || exit $?

# Start a container.
docker run \
    --name "${MUTAGEN_TEST_DOCKER_CONTAINER_NAME}" \
    --detach \
    "${MUTAGEN_TEST_DOCKER_IMAGE_NAME}" || exit $?
