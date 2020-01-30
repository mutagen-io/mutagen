#!/bin/sh

# Wait for the development volume to be populated.
while [ ! -f /development/ready ]; do
    echo "Waiting for code to be populated..."
    sleep 1
done

# Switch to the API server source directory.
cd /development/code/api

# Build the API server.
echo "Building API server..."
go build -o api-server .

# Run the API server.
echo "Starting API server..."
./api-server
