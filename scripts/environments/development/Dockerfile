# Use a minimal base image with Go support.
FROM golang:alpine

# Install supplementary tools.
RUN ["apk", "add", "git", "openssh-client"]
RUN ["go", "get", "golang.org/x/tools/cmd/godoc"]

# Disable cgo.
ENV CGO_ENABLED=0

# Create a directory that will serve as our working directory, as well as a
# synchronization root.
RUN ["mkdir", "/root/mutagen"]

# Set the working directory.
WORKDIR /root/mutagen

# Use a null entry point to just idle.
ENTRYPOINT ["tail", "-f", "/dev/null"]
