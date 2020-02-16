#!/bin/sh

# Switch to the web server source directory.
cd /code/web

# Build the web server.
echo "Building web server..."
go build -o web-server .

# Run the web server.
echo "Starting web server..."
./web-server
