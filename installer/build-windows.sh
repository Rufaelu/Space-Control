#!/bin/sh
# Builds the Windows game and its setup program.
# Run inside an MSYS2 MINGW64 shell on Windows, after:
#   pacman -S mingw-w64-x86_64-gcc mingw-w64-x86_64-go mingw-w64-x86_64-SDL2 mingw-w64-x86_64-nsis
# The game links MSYS2's SDL2 dynamically; SDL2.dll and any other MinGW DLLs it
# needs are copied next to the exe so the installed game is self-contained.
# (go-sdl2's bundled static SDL2 no longer links with current MinGW toolchains.)
set -eu
cd "$(dirname "$0")/.."
mkdir -p build/windows dist
CGO_ENABLED=1 go build -trimpath -ldflags "-s -w -H windowsgui" -o build/windows/SpaceControl.exe .
cp README.md build/windows/README.txt

# ship every DLL the exe pulls in from the MinGW tree, including indirect ones
PREFIX="${MINGW_PREFIX:-/mingw64}"
ldd build/windows/SpaceControl.exe | awk '{print $3}' | grep -i "^$PREFIX/bin/" | sort -u |
while read -r dll; do
    echo "bundling $(basename "$dll")"
    cp "$dll" build/windows/
done
# if MSYS2's SDL2 is the sdl2-compat shim, it loads SDL3.dll at run time, which ldd can't see
if [ -f "$PREFIX/bin/SDL3.dll" ]; then
    for dll in "$PREFIX/bin/SDL3.dll" $(ldd "$PREFIX/bin/SDL3.dll" | awk '{print $3}' | grep -i "^$PREFIX/bin/"); do
        echo "bundling $(basename "$dll")"
        cp "$dll" build/windows/
    done
fi
if [ ! -f build/windows/SDL2.dll ]; then
    echo "SDL2.dll was not found among the exe's dependencies" >&2
    exit 1
fi

makensis installer/windows/SpaceControl.nsi
echo "Built dist/SpaceControl-windows-x86_64-setup.exe"
