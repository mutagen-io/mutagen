go build -o scripts\docker_windows\httpdemo.exe github.com/havoc-io/mutagen/pkg/integration/fixtures/httpdemo
docker version
docker image build --tag %MUTAGEN_TEST_DOCKER_IMAGE_NAME% scripts\docker_windows
del /q scripts\docker_windows\httpdemo.exe
docker container run --name %MUTAGEN_TEST_DOCKER_CONTAINER_NAME% --detach %MUTAGEN_TEST_DOCKER_IMAGE_NAME%
