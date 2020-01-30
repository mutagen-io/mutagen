# Use a minimal base image.
FROM alpine:latest

# Create a directory to store the code snapshot.
RUN ["mkdir", "-p", "/snapshot/code"]

# Copy in code directories to the snapshot. Note that the build context for this
# Dockerfile is the repository itself. We have to do each copy separately
# because Docker's COPY directive doesn't work like mv (where multiple
# directories can be copied into a target directory).
COPY ["api", "/snapshot/code/api"]
COPY ["frontend", "/snapshot/code/frontend"]
COPY ["web", "/snapshot/code/web"]

# Copy in the entry point script.
COPY ["development/entrypoint.sh", "/entrypoint.sh"]

# Ensure that the entry point script is executable.
RUN ["chmod", "+x", "/entrypoint.sh"]

# Set the entrypoint.
ENTRYPOINT /entrypoint.sh
