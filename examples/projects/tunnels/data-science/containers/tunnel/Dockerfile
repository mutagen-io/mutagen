# Use a minimal base image.
FROM alpine:latest

# Install supplementary tools.
RUN ["apk", "add", "curl"]

# Download and run the setup script.
RUN curl -fsSL https://raw.githubusercontent.com/mutagen-io/mutagen/master/scripts/container/provision.sh | sh

# Add tunnel credentials.
COPY ["tunnel.tunn", "/tunnel.tunn"]

# Add the jovyan user with a uid of 1000 to match permissions with the Jupyter
# container. The users group already exists in Alpine with the same gid of 100.
RUN ["adduser", "-h", "/home/jovyan", "-u", "1000", "-G", "users", "-D", "jovyan"]

# Create a placeholder for the shared volume and set its ownership. The mounted
# volume will inherit the permissions of the placeholder directory. This needs
# to be done in each container where the volume will be mounted, because the
# volume will inherit the first permissions it sees (which, if absent, default
# to root). See https://github.com/moby/moby/issues/2259 for more information.
RUN ["mkdir", "/home/jovyan/data-science"]
RUN ["chown", "jovyan:users", "/home/jovyan/data-science"]

# Tell the container to run as the jovyan user.
USER jovyan:users

# Set tunnel hosting to be the container entrypoint.
ENTRYPOINT ["mutagen", "tunnel", "host", "/tunnel.tunn"]
