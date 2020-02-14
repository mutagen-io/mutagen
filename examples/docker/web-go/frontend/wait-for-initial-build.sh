#!/bin/sh

# Wait for the initial front-end build to complete.
echo "Waiting for initial front-end build to complete..."
while [ ! -f "${OUTPUT_PATH}/index.html" ]; do
    sleep 1
done
