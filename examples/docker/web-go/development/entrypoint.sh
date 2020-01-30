#!/bin/sh

# Check if the code volume has been populated (as indicated by the presence of
# an indicator file). If not, then copy the code from our snapshot.
if [ ! -f /development/ready ]; then
    echo "Populating development volume from snapshot..."
    cp -R /snapshot/code /development/code
    echo "Development volume populated"
    touch /development/ready
fi

# Run a no-op entry point and wait to host Mutagen agent processes.
tail -f /dev/null
