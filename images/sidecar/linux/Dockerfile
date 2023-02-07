# Use an Alpine-based Go builder.
FROM golang:1.20.0-alpine3.17 AS builder

# Disable cgo in order to match the behavior of our release binaries (and to
# avoid the need for gcc on certain architectures).
ENV CGO_ENABLED=0

# Copy the Mutagen source code into the container and set the code directory as
# our default working location.
RUN ["mkdir", "/mutagen"]
COPY ["go.mod", "go.sum", "/mutagen/"]
COPY ["cmd", "/mutagen/cmd/"]
COPY ["pkg", "/mutagen/pkg/"]
COPY ["sspl", "/mutagen/sspl/"]
WORKDIR /mutagen

# Build the sidecar entry point and agent binaries.
RUN ["go", "build", "./cmd/mutagen-sidecar"]
RUN ["go", "build", "-tags", "sspl", "./cmd/mutagen-agent"]
RUN ["go", "build", "-o", "mutagen-agent-mit", "./cmd/mutagen-agent"]


# Switch to a vanilla Alpine base for the final image.
FROM alpine:3.16 AS base

# Copy the sidecar entry point from the builder.
COPY --from=builder ["/mutagen/mutagen-sidecar", "/usr/bin/mutagen-sidecar"]

# Create the parent directory for volume mount points.
RUN ["mkdir", "/volumes"]

# Add an indicator that this is a Mutagen sidecar container.
ENV MUTAGEN_SIDECAR=1

# Set the image entry point.
ENTRYPOINT ["mutagen-sidecar"]


# Define the standard sidecar image.
FROM base as standard

# Copy the standard agent from the builder and use its installation mechanism to
# move it to the correct location.
COPY --from=builder ["/mutagen/mutagen-agent", "/mutagen-agent"]
RUN ["/mutagen-agent", "install"]


# Define the MIT-only sidecar image.
FROM base as mit

# Copy the MIT-only agent from the builder and use its installation mechanism to
# move it to the correct location.
COPY --from=builder ["/mutagen/mutagen-agent-mit", "/mutagen-agent"]
RUN ["/mutagen-agent", "install"]
