#!/bin/sh
# Builds the game and packs dist/SpaceControl-linux-x86_64-installer.run
# Needs: Go, a C compiler, SDL2 development files.
#
# BUNDLE_SDL2=1 ships libSDL2 inside the game folder, so the player needs
# nothing installed. installer/build-installers.sh does that inside an
# Ubuntu 22.04 container, so the result runs on that and every newer distro;
# without it the installed game uses the system's SDL2.
set -eu
cd "$(dirname "$0")/.."
rm -rf build/linux
mkdir -p build/linux/music dist
if [ "${BUNDLE_SDL2:-0}" = 1 ]; then
    # $ORIGIN/lib: look for libSDL2 in the lib folder beside the executable
    go build -trimpath -ldflags '-s -w -extldflags "-Wl,-rpath,$ORIGIN/lib"' -o build/linux/SpaceControl .
    mkdir -p build/linux/lib
    cp -L "$(pkg-config --variable=libdir sdl2)/libSDL2-2.0.so.0" build/linux/lib/
else
    go build -trimpath -ldflags "-s -w" -o build/linux/SpaceControl .
fi
cp README.md build/linux/README.md
cp music/README.txt build/linux/music/README.txt
OUT=dist/SpaceControl-linux-x86_64-installer.run
cat installer/linux/install-stub.sh > "$OUT"
tar -cz -C build/linux . >> "$OUT"
chmod +x "$OUT"
echo "Built $OUT ($(du -h "$OUT" | cut -f1))"
