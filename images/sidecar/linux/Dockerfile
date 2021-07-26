# Use an Alpine-based Go builder.
# TODO: It would be better to use a tag with more precise version information
# here (ideally explicitly matching the Alpine version specified below), but for
# some reason there are inconsistencies between the available architectures with
# different Go image tag formats (even though the tags are equivalent). More
# specifically, a linux/s390x image doesn't seem to be available for tag formats
# with more precise version information. For now we'll just use the more general
# tag format (and manually check version alignment), but we should use the more
# precise tags once the inconsistencies are resolved.
FROM golang:1.16-alpine AS builder

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
