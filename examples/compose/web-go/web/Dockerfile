# Use a minimal base image with Go support.
FROM golang:alpine

# Copy in the entry point script and ensure that it's executable.
COPY ["entrypoint.sh", "/entrypoint.sh"]
RUN ["chmod", "+x", "/entrypoint.sh"]

# Set the entrypoint.
ENTRYPOINT ["/entrypoint.sh"]
