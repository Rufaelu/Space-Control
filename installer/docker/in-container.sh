#!/bin/sh
# Runs inside a build container started by installer/build-installers.sh.
# The source is mounted read-only at /src; this copies it, builds one
# installer, and writes the finished dist/ folder to stdout as a tar stream
# (so the files land on the host owned by whoever ran the build). Build
# output goes to stderr.
#
#   sh in-container.sh linux|windows
set -eu
exec 3>&1 1>&2

export GOMODCACHE=/cache/mod GOCACHE=/cache/build GOFLAGS=-buildvcs=false
mkdir -p /work
tar -C /src --exclude=./.git --exclude=./build --exclude=./dist --exclude='./GamePlay.*' -c . | tar -C /work -x
cd /work

case "${1:-}" in
    linux) BUNDLE_SDL2=1 sh installer/build-linux.sh ;;
    windows) sh installer/docker/build-windows-cross.sh ;;
    *) echo "usage: in-container.sh linux|windows" >&2; exit 2 ;;
esac

tar -C dist -c . >&3
