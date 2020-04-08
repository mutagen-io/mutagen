#!/bin/sh

# Switch to the API server source directory.
cd /code/api

# Build the API server.
echo "Building API server..."
go build -o api-server .

# Run the API server. We use exec to replace the shell process so that the
# server receives termination signals.
echo "Starting API server..."
exec ./api-server
