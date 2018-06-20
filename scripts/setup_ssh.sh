#!/bin/bash

# SSH installation/activation based on the OS.
if [[ "$TRAVIS_OS_NAME" == "osx" ]]; then
    sudo systemsetup -f -setremotelogin on
elif [[ "$TRAVIS_OS_NAME" == "linux" ]]; then
    sudo apt-get -qq install openssh-client openssh-server
    sudo service ssh restart
fi

# Generate an SSH key in quiet mode without a password.
ssh-keygen -q -t rsa -b 4096 -N ""

# Add the key to our list of authorized keys.
cat ~/.ssh/id_rsa.pub >> ~/.ssh/authorized_keys

# Add localhost to our list of known hosts (so we're not prompted).
ssh-keyscan -t rsa localhost >> ~/.ssh/known_hosts

# Mark the environment as having SSH support.
export MUTAGEN_TEST_SSH="true"
