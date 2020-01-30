# Use a minimal base image with Node.js support.
FROM node:alpine

# Copy in the entry point script.
COPY ["entrypoint.sh", "/entrypoint.sh"]

# Ensure that the entry point script is executable.
RUN ["chmod", "+x", "/entrypoint.sh"]

# Set the entrypoint.
ENTRYPOINT /entrypoint.sh
