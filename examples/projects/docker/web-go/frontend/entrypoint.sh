#!/bin/sh

# Switch to the frontend directory.
cd /code/frontend

# Perform a global gulp installation and an npm install operation if needed.
if [ ! -d node_modules ]; then
    echo "Installing npm modules..."
    npm install gulp-cli -g
    npm install || exit 1
    echo "npm install complete"
fi

# Run gulp. We use exec to replace the shell process so that gulp receives
# termination signals.
exec gulp
