#!/bin/bash

# Exit immediately on failure.
set -e

# Determine the operating system.
MUTAGEN_OS_NAME="$(go env GOOS)"

# Verify that the platform is supported and that its SSH server is enabled.
if [[ "${MUTAGEN_OS_NAME}" == "darwin" ]]; then
    sudo systemsetup -f -setremotelogin on
elif [[ "${MUTAGEN_OS_NAME}" == "linux" ]]; then
    sudo apt-get -qq install openssh-client openssh-server
    sudo service ssh restart
else
    # TODO: We should be able to support Windows 10 via its OpenSSH server.
    echo "SSH server CI setup not supported on this platform" 1>&2
    exit 1
fi

# Ensure that home directory permissions are acceptable to sshd.
chmod 755 ~

# Ensure our SSH configuration directory exists and has the correct permissions.
mkdir -p ~/.ssh
chmod 700 ~/.ssh

# Generate an SSH key in quiet mode without a password. We don't need to set
# permissions explicitly since ssh-keygen will set them correctly.
ssh-keygen -q -t ed25519 -C "ci@localhost" -N "" -f ~/.ssh/id_ed25519

# Add the key to our list of authorized keys and ensure that the authorized keys
# file has the correct permissions.
cat ~/.ssh/id_ed25519.pub >> ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys

# Add localhost to our list of known hosts (so we're not prompted) and ensure
# that the known hosts file has the correct permissions.
ssh-keyscan -t ed25519 localhost >> ~/.ssh/known_hosts
chmod 644 ~/.ssh/known_hosts
