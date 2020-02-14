#!/bin/sh

# Switch to the API server source directory.
cd /code/web-go/api

# Build the API server.
echo "Building API server..."
go build -o api-server .

# Run the API server.
echo "Starting API server..."
./api-server
