docker version
docker build --tag %MUTAGEN_TEST_DOCKER_IMAGE_NAME% --file scripts/dockerfile_windows scripts
docker run --name %MUTAGEN_TEST_DOCKER_CONTAINER_NAME% --detach %MUTAGEN_TEST_DOCKER_IMAGE_NAME%
