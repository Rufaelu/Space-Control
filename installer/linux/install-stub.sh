#!/bin/sh
# Space Control installer for Linux. Self-extracting: the game files are
# appended to this script as a tar.gz archive after the __PAYLOAD__ line.
#
#   sh SpaceControl-linux-x86_64-installer.run [--uninstall]
# The game asks for the pilot's name itself the first time it starts.
set -eu

: "${HOME:?HOME is not set}"
APP=space-control
DATA="${XDG_DATA_HOME:-$HOME/.local/share}/$APP"
CONF="${XDG_CONFIG_HOME:-$HOME/.config}/$APP"
LINK="$HOME/.local/bin/$APP"
DESKTOP="${XDG_DATA_HOME:-$HOME/.local/share}/applications/$APP.desktop"

# remove_install deletes the game's own folder, launcher link and menu entry,
# but keeps any songs the player put in the music folder.
remove_install() {
    case "$DATA" in */"$APP") ;; *) echo "Unexpected install folder: $DATA" >&2; exit 1 ;; esac
    rm -f -- "${LINK:?}" "${DESKTOP:?}"
    if [ -d "$DATA" ]; then
        for f in "$DATA"/* "$DATA"/.[!.]*; do
            [ -e "$f" ] || [ -L "$f" ] || continue
            [ "$f" = "$DATA/music" ] || rm -rf -- "$f"
        done
        rm -f -- "$DATA/music/README.txt"
        rmdir -- "$DATA/music" 2>/dev/null || true
        rmdir -- "$DATA" 2>/dev/null || true
    fi
    echo "Space Control removed. Your name and high score are kept in $CONF"
    if [ -d "$DATA/music" ]; then
        echo "Your songs are kept in $DATA/music"
    fi
}

while [ $# -gt 0 ]; do
    case "$1" in
        --uninstall) remove_install; exit 0 ;;
        -h|--help) sed -n '2,6p' "$0"; exit 0 ;;
        *) echo "Unknown option: $1" >&2; exit 2 ;;
    esac
done

echo
echo "  ============================="
echo "    S P A C E   C O N T R O L"
echo "  ============================="
echo

# --- unpack (songs already in the music folder are left alone) ---
mkdir -p "$DATA" "$(dirname "$LINK")" "$(dirname "$DESKTOP")"
LINE=$(awk '/^__PAYLOAD__$/ { print NR + 1; exit }' "$0")
tail -n +"$LINE" "$0" | tar -xz -C "$DATA"
chmod +x "$DATA/SpaceControl"
mkdir -p "$DATA/music"

# --- launcher, menu entry, uninstaller ---
ln -sf "$DATA/SpaceControl" "$LINK"
cat > "$DESKTOP" <<DESKTOP_EOF
[Desktop Entry]
Type=Application
Name=Space Control
Comment=3D space shooter
Exec="$DATA/SpaceControl"
Icon=applications-games
Terminal=false
Categories=Game;ActionGame;
Actions=music;

[Desktop Action music]
Name=Open Music Folder
Exec=xdg-open "$DATA/music"
DESKTOP_EOF
# the uninstaller is this same script, run with --uninstall
cp "$0" "$DATA/installer.run"
printf '#!/bin/sh\nexec sh "%s/installer.run" --uninstall\n' "$DATA" > "$DATA/uninstall.sh"
chmod +x "$DATA/uninstall.sh"

echo "  Installed. The game asks for your pilot name when it first starts."
echo "    Game:       $DATA/SpaceControl"
echo "    Start it:   $APP   (or find Space Control in your applications menu)"
echo "    Your music: $DATA/music"
echo "                (put MP3, OGG or WAV songs there and the game plays them)"
echo "    Uninstall:  $DATA/uninstall.sh"
case ":$PATH:" in
    *":$(dirname "$LINK"):"*) ;;
    *) echo "    Note: $(dirname "$LINK") is not on your PATH; use the menu entry or the full path." ;;
esac

# --- the installer normally carries SDL2; an unbundled build needs the system's ---
if [ ! -f "$DATA/lib/libSDL2-2.0.so.0" ] && ! { ldconfig -p 2>/dev/null | grep -q 'libSDL2-2\.0\.so\.0'; }; then
    echo
    echo "  SDL2 does not seem to be installed. Install it before playing:"
    echo "    Debian/Ubuntu:  sudo apt install libsdl2-2.0-0"
    echo "    Fedora:         sudo dnf install SDL2"
    echo "    Arch:           sudo pacman -S sdl2-compat"
fi
# OpenGL comes with the graphics driver; only a bare system lacks it
if ! { ldconfig -p 2>/dev/null | grep -q 'libGL\.so\.1'; }; then
    echo
    echo "  OpenGL (libGL.so.1) was not found. Install your graphics drivers, e.g.:"
    echo "    Debian/Ubuntu:  sudo apt install libgl1 libgl1-mesa-dri"
    echo "    Fedora:         sudo dnf install mesa-libGL mesa-dri-drivers"
fi
echo
exit 0
__PAYLOAD__
