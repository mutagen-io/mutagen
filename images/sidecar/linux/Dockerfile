# Use an Alpine-based Go builder.
FROM golang:1.16-alpine3.14 AS builder

# Disable cgo in order to match the behavior of our release binaries (and to
# avoid the need for gcc on certain architectures).
ENV CGO_ENABLED=0

# Define an argument to specify the entrypoint version target.
ARG TARGET

# Build the sidecar entrypoint. We have to use the shell form of RUN in order to
# access the TARGET argument.
RUN go get github.com/mutagen-io/mutagen/cmd/mutagen-sidecar@${TARGET}

# Switch to a vanilla Alpine base for the final image. In theory, we could also
# use scratch as our base here, but Alpine provides the ability to drop into a
# shell, which can be useful for debugging.
FROM alpine:3.14

# Copy the sidecar entrypoint from the builder.
COPY --from=builder ["/go/bin/mutagen-sidecar", "/usr/bin/mutagen-sidecar"]

# Create the parent directory for volume mount points.
RUN ["mkdir", "/volumes"]

# Set the image entrypoint.
ENTRYPOINT ["mutagen-sidecar"]
