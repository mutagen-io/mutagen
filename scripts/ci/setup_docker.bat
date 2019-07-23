go build -o scripts\ci\docker\windows\httpdemo.exe github.com/mutagen-io/mutagen/pkg/integration/fixtures/httpdemo
docker version
docker image build --tag %MUTAGEN_TEST_DOCKER_IMAGE_NAME% scripts\ci\docker\windows
del /q scripts\ci\docker\windows\httpdemo.exe
docker container run --name %MUTAGEN_TEST_DOCKER_CONTAINER_NAME% --detach %MUTAGEN_TEST_DOCKER_IMAGE_NAME%
