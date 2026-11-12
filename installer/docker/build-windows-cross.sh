#!/bin/sh
# Cross-builds the Windows game and its setup program on Linux. It runs inside
# the Windows build container (installer/build-installers.sh starts it), which
# has MinGW-w64, the official SDL2 MinGW development release and NSIS.
# (installer/build-windows.sh is the same job for an MSYS2 shell on Windows.)
set -eu
cd "$(dirname "$0")/../.."
SDL="${SDL2_MINGW:?set SDL2_MINGW to SDL2's x86_64-w64-mingw32 folder}"
CC=x86_64-w64-mingw32-gcc
OBJDUMP=x86_64-w64-mingw32-objdump

rm -rf build/windows
mkdir -p build/windows/music dist
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC="$CC" \
    CGO_CFLAGS="-O2 -I$SDL/include" CGO_LDFLAGS="-L$SDL/lib" \
    go build -trimpath -ldflags "-s -w -H windowsgui" -o build/windows/SpaceControl.exe .
cp "$SDL/bin/SDL2.dll" build/windows/
cp README.md build/windows/README.txt
cp music/README.txt build/windows/music/README.txt

# Every DLL the game loads must ship with it or come with Windows. Ship any
# that the MinGW toolchain provides; the rest must be on the known list.
GCCDIR=$(dirname "$("$CC" -print-libgcc-file-name)")
ship() { # name: copy a MinGW runtime DLL if the toolchain has one
    for d in "$GCCDIR" /usr/x86_64-w64-mingw32/lib /usr/x86_64-w64-mingw32/bin; do
        if [ -f "$d/$1" ]; then
            echo "bundling $1"
            cp "$d/$1" build/windows/
            return 0
        fi
    done
    return 1
}
for bin in build/windows/SpaceControl.exe build/windows/SDL2.dll; do
    "$OBJDUMP" -p "$bin" | awk '/DLL Name:/ {print $3}'
done | sort -u | while read -r dll; do
    lower=$(printf '%s' "$dll" | tr 'A-Z' 'a-z')
    case "$lower" in
        sdl2.dll) ;;
        kernel32.dll | user32.dll | gdi32.dll | winmm.dll | imm32.dll | ole32.dll | oleaut32.dll | \
        shell32.dll | version.dll | setupapi.dll | advapi32.dll | ws2_32.dll | msvcrt.dll | \
        opengl32.dll | cfgmgr32.dll | hid.dll | dinput8.dll | xinput1_4.dll | \
        api-ms-win-*.dll) ;;
        *) ship "$dll" || { echo "the game needs $dll, which is neither bundled nor part of Windows" >&2; exit 1; } ;;
    esac
done

makensis -V2 installer/windows/SpaceControl.nsi
echo "Built dist/SpaceControl-windows-x86_64-setup.exe ($(du -h dist/SpaceControl-windows-x86_64-setup.exe | cut -f1))"
