# Use a minimal base image with Node.js support.
FROM node:alpine

# Copy in the entry point script and ensure that it's executable.
COPY ["entrypoint.sh", "/entrypoint.sh"]
RUN ["chmod", "+x", "/entrypoint.sh"]

# Copy in the initial build wait script and ensure that it's executable.
COPY ["wait-for-initial-build.sh", "/wait-for-initial-build.sh"]
RUN ["chmod", "+x", "/wait-for-initial-build.sh"]

# Set the entrypoint.
ENTRYPOINT ["/entrypoint.sh"]
