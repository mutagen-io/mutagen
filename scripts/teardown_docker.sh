#!/bin/bash

# Stop our test container.
docker stop "${MUTAGEN_TEST_DOCKER_CONTAINER_NAME}" || exit $?

# Prune containers.
docker container prune --force || exit $?

# Remove our images.
docker image rm --force "${MUTAGEN_TEST_DOCKER_IMAGE_NAME}" || exit $?
docker image rm --force "${MUTAGEN_TEST_DOCKER_BASE_IMAGE_NAME}" || exit $?
