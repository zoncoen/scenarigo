#!/bin/bash -e
# This file is used in the golang-cross container.

if [ -d "/scenarigo" ]; then
    git config --global --add safe.directory /scenarigo
fi


# TODO: --rm-dist is deprecated, use --clean
cmd="bash /entrypoint.sh --skip-publish --rm-dist --debug"
if [ "${SNAPSHOT}" != "" ]; then
    echo "snapshot build"
    cmd="${cmd} --snapshot"
fi
eval ${cmd}
