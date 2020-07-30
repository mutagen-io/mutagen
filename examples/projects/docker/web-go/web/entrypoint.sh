#!/bin/sh

# Switch to the web server source directory.
cd /code/web

# Build the web server.
echo "Building web server..."
go build -o web-server .

# Run the web server. We use exec to replace the shell process so that the
# server receives termination signals.
echo "Starting web server..."
exec ./web-server
