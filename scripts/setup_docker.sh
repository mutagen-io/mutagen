#!/bin/bash

# Set up parameters.
export MUTAGEN_TEST_DOCKER_BASE_IMAGE_NAME="alpine"
export MUTAGEN_TEST_DOCKER_IMAGE_NAME="mutagentest"
export MUTAGEN_TEST_DOCKER_CONTAINER_NAME="mutagentester"

# Pull our base Docker image.
docker pull "${MUTAGEN_TEST_DOCKER_BASE_IMAGE_NAME}" || exit $?

# Build our image.
docker build \
    --tag "${MUTAGEN_TEST_DOCKER_IMAGE_NAME}" \
    --file scripts/dockerfile_linux || exit $?

# Start a container.
docker run \
    --name "${MUTAGEN_TEST_DOCKER_CONTAINER_NAME}" \
    --detach \
    "${MUTAGEN_TEST_DOCKER_IMAGE_NAME}" || exit $?

# Mark the environment as having Docker support.
export MUTAGEN_TEST_DOCKER="true"
