#!/bin/sh
set -e

# Execute dockerd-entrypoint.sh
./dockerd-entrypoint.sh > /dev/null 2>&1 &

# Execute app
./app

# Start the main process
exec "$@"
