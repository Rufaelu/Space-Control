#!/bin/sh
# Builds the Windows game and its setup program.
# Run inside an MSYS2 MINGW64 shell on Windows, after:
#   pacman -S mingw-w64-x86_64-gcc mingw-w64-x86_64-go mingw-w64-x86_64-nsis
# SDL2 is linked statically from the copy bundled with go-sdl2, so no SDL2.dll ships.
set -eu
cd "$(dirname "$0")/.."
mkdir -p build/windows dist
CGO_ENABLED=1 go build -trimpath -tags static -ldflags "-s -w -H windowsgui" -o build/windows/SpaceControl.exe .
cp README.md build/windows/README.txt
# ship any MinGW runtime DLLs the exe still depends on
for dll in $(objdump -p build/windows/SpaceControl.exe | awk '/DLL Name/ {print $3}'); do
    if [ -f "${MINGW_PREFIX:-/mingw64}/bin/$dll" ]; then
        cp "${MINGW_PREFIX:-/mingw64}/bin/$dll" build/windows/
    fi
done
makensis installer/windows/SpaceControl.nsi
echo "Built dist/SpaceControl-windows-x86_64-setup.exe"
