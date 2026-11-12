#!/bin/sh
# Builds the installers you copy to another PC. They carry everything the
# game needs, so nothing has to be installed on that PC first:
#
#   dist/SpaceControl-windows-x86_64-setup.exe     Windows 10/11, 64-bit
#   dist/SpaceControl-linux-x86_64-installer.run   64-bit desktop Linux from 2022 on
#   dist/HOW-TO-INSTALL.txt
#
# Needs only Docker. The first run downloads and builds the toolchains, which
# takes a few minutes; later runs reuse them.
#
#   sh installer/build-installers.sh             # both
#   sh installer/build-installers.sh windows     # or just one: linux | windows
set -eu
cd "$(dirname "$0")/.."
targets="${*:-linux windows}"
for t in $targets; do
    case "$t" in
        linux | windows) ;;
        *) echo "unknown target '$t' (use linux or windows)" >&2; exit 2 ;;
    esac
done
command -v docker >/dev/null || { echo "Docker is needed: https://docs.docker.com/get-docker/" >&2; exit 1; }
if ! docker info >/dev/null 2>&1; then
    # Docker Desktop on Linux runs as a user service; start it and wait for it
    if systemctl --user cat docker-desktop.service >/dev/null 2>&1; then
        echo "==> starting Docker Desktop"
        systemctl --user start docker-desktop
        i=0
        until docker info >/dev/null 2>&1; do
            i=$((i + 1))
            [ "$i" -le 60 ] || { echo "Docker Desktop did not come up; start it and try again." >&2; exit 1; }
            sleep 2
        done
    else
        echo "Docker is installed but not running; start it and try again." >&2
        exit 1
    fi
fi

mkdir -p dist
tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT
for t in $targets; do
    img="space-control-build-$t"
    echo "==> $t: preparing the build environment"
    if ! docker build -t "$img" -f "installer/docker/$t.Dockerfile" installer/docker; then
        # e.g. offline: Docker could not check the base image for updates
        docker image inspect "$img" >/dev/null 2>&1 || exit 1
        echo "    (could not refresh it; using the build environment from last time)"
    fi
    echo "==> $t: building"
    docker run --rm -v "$PWD:/src:ro" -v space-control-gocache:/cache "$img" \
        sh /src/installer/docker/in-container.sh "$t" > "$tmp"
    tar -x -C dist -f "$tmp"
done
cp installer/HOW-TO-INSTALL.txt dist/
echo
echo "Done. Copy these to the other PC (a USB stick works):"
for f in dist/*; do
    echo "  $f  ($(du -h "$f" | cut -f1))"
done
