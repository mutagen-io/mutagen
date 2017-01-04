#!/bin/sh

# Convert the 386 bundle.
tar xzf build/mutagen_windows_386.tar.gz
zip build/mutagen_windows_386.zip mutagen.exe mutagen-agents.tar.gz
rm mutagen.exe mutagen-agents.tar.gz

# Convert the amd64 bundle.
tar xzf build/mutagen_windows_amd64.tar.gz
zip build/mutagen_windows_amd64.zip mutagen.exe mutagen-agents.tar.gz
rm mutagen.exe mutagen-agents.tar.gz
