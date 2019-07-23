#!/bin/bash

# SSH installation/activation based on the OS.
if [[ "$TRAVIS_OS_NAME" == "osx" ]]; then
    sudo systemsetup -f -setremotelogin on || exit $?
elif [[ "$TRAVIS_OS_NAME" == "linux" ]]; then
    sudo apt-get -qq install openssh-client openssh-server || exit $?
    sudo service ssh restart || exit $?
fi

# Ensure our SSH configuration directory exists.
mkdir -p ~/.ssh || exit $?

# Generate an SSH key in quiet mode without a password.
ssh-keygen -q -t rsa -b 4096 -N "" -f ~/.ssh/id_rsa || exit $?

# Add the key to our list of authorized keys.
cat ~/.ssh/id_rsa.pub >> ~/.ssh/authorized_keys || exit $?

# Add localhost to our list of known hosts (so we're not prompted).
ssh-keyscan -t rsa localhost >> ~/.ssh/known_hosts || exit $?
