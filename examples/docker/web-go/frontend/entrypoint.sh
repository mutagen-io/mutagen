#!/bin/sh

# Wait for the development volume to be populated.
while [ ! -f /development/ready ]; do
    echo "Waiting for code to be populated..."
    sleep 1
done

# Switch to the frontend directory.
cd /development/code/frontend

# Perform a global gulp installation and an npm install operation if needed.
if [ ! -d node_modules ]; then
    echo "Installing npm modules..."
    npm install gulp-cli -g
    npm install || exit 1
    echo "npm install complete"
fi

# Run gulp.
gulp
