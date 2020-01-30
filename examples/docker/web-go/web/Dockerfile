# Use a minimal base image with Go support.
FROM golang:alpine

# Disable cgo.
ENV CGO_ENABLED=0

# Copy in the entry point script.
COPY ["entrypoint.sh", "/entrypoint.sh"]

# Ensure that the entry point script is executable.
RUN ["chmod", "+x", "/entrypoint.sh"]

# Set the entrypoint.
ENTRYPOINT /entrypoint.sh
