#!/bin/sh
# Builds the game and packs dist/SpaceControl-linux-x86_64-installer.run
# Needs: Go, a C compiler, SDL2 development files.
set -eu
cd "$(dirname "$0")/.."
mkdir -p build/linux dist
go build -trimpath -ldflags "-s -w" -o build/linux/SpaceControl .
cp README.md build/linux/README.md
OUT=dist/SpaceControl-linux-x86_64-installer.run
cat installer/linux/install-stub.sh > "$OUT"
tar -cz -C build/linux SpaceControl README.md >> "$OUT"
chmod +x "$OUT"
echo "Built $OUT ($(du -h "$OUT" | cut -f1))"
