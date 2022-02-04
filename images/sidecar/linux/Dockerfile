# Use an Alpine-based Go builder.
FROM golang:1.17-alpine3.14 AS builder

# Disable cgo in order to match the behavior of our release binaries (and to
# avoid the need for gcc on certain architectures).
ENV CGO_ENABLED=0

# Copy the Mutagen source code into the container and set the code directory as
# our default working location.
RUN ["mkdir", "/mutagen"]
COPY ["go.mod", "go.sum", "/mutagen/"]
COPY ["cmd", "/mutagen/cmd/"]
COPY ["pkg", "/mutagen/pkg/"]
WORKDIR /mutagen

# Build the sidecar entrypoint and agent binaries.
RUN ["go", "build", "./cmd/mutagen-sidecar"]
RUN ["go", "build", "./cmd/mutagen-agent"]

# Switch to a vanilla Alpine base for the final image.
FROM alpine:3.14

# Copy the sidecar entrypoint and agent binaries from the builder. For the
# agent, use its installation mechanism to move it to the correct location.
COPY --from=builder ["/mutagen/mutagen-sidecar", "/usr/bin/mutagen-sidecar"]
COPY --from=builder ["/mutagen/mutagen-agent", "/mutagen-agent"]
RUN ["/mutagen-agent", "install"]

# Create the parent directory for volume mount points.
RUN ["mkdir", "/volumes"]

# Add an indicator that this is a Mutagen sidecar container.
ENV MUTAGEN_SIDECAR=1

# Set the image entrypoint.
ENTRYPOINT ["mutagen-sidecar"]
