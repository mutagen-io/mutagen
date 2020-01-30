#!/bin/sh

# Wait for the development volume to be populated.
while [ ! -f /development/ready ]; do
    echo "Waiting for code to be populated..."
    sleep 1
done

# Switch to the web server source directory.
cd /development/code/web

# Build the web server.
echo "Building web server..."
go build -o web-server .

# Run the web server.
echo "Starting web server..."
./web-server
