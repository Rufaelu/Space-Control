#!/bin/sh
# Space Control installer for Linux. Self-extracting: the game files are
# appended to this script as a tar.gz archive after the __PAYLOAD__ line.
#
#   sh SpaceControl-linux-x86_64-installer.run [--name YOUR_NAME] [--uninstall]
set -eu

: "${HOME:?HOME is not set}"
APP=space-control
DATA="${XDG_DATA_HOME:-$HOME/.local/share}/$APP"
CONF="${XDG_CONFIG_HOME:-$HOME/.config}/$APP"
LINK="$HOME/.local/bin/$APP"
DESKTOP="${XDG_DATA_HOME:-$HOME/.local/share}/applications/$APP.desktop"

# remove_install deletes only the game's own folder, launcher link and menu entry.
remove_install() {
    case "$DATA" in */"$APP") rm -rf -- "${DATA:?}" ;; esac
    rm -f -- "${LINK:?}" "${DESKTOP:?}"
    echo "Space Control removed. Your name and high score are kept in $CONF"
}

NAME=""
while [ $# -gt 0 ]; do
    case "$1" in
        --name) NAME="${2:-}"; shift 2 ;;
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

# --- ask for the pilot's name (used in-game to congratulate them) ---
if [ -z "$NAME" ]; then
    if [ -r /dev/tty ] && [ -t 1 ]; then
        printf "  What is your name, pilot? "
        read -r NAME < /dev/tty || NAME=""
    fi
    [ -n "$NAME" ] || NAME="${USER:-PILOT}"
fi
# keep what the game's font can draw: letters, digits, space, dash; max 12
NAME=$(printf '%s' "$NAME" | tr '[:lower:]' '[:upper:]' | tr -cd 'A-Z0-9 -' | cut -c1-12)
[ -n "$NAME" ] || NAME=PILOT

# --- unpack ---
mkdir -p "$DATA" "$CONF" "$(dirname "$LINK")" "$(dirname "$DESKTOP")"
LINE=$(awk '/^__PAYLOAD__$/ { print NR + 1; exit }' "$0")
tail -n +"$LINE" "$0" | tar -xz -C "$DATA"
chmod +x "$DATA/SpaceControl"
printf '%s\n' "$NAME" > "$CONF/name"

# --- launcher, menu entry, uninstaller ---
ln -sf "$DATA/SpaceControl" "$LINK"
cat > "$DESKTOP" <<DESKTOP_EOF
[Desktop Entry]
Type=Application
Name=Space Control
Comment=3D space shooter
Exec=$DATA/SpaceControl
Icon=applications-games
Terminal=false
Categories=Game;ActionGame;
DESKTOP_EOF
# the uninstaller is this same script, run with --uninstall
cp "$0" "$DATA/installer.run"
printf '#!/bin/sh\nexec sh "%s/installer.run" --uninstall\n' "$DATA" > "$DATA/uninstall.sh"
chmod +x "$DATA/uninstall.sh"

echo "  Installed for $NAME."
echo "    Game:       $DATA/SpaceControl"
echo "    Start it:   $APP   (or find Space Control in your applications menu)"
echo "    Uninstall:  $DATA/uninstall.sh"
case ":$PATH:" in
    *":$(dirname "$LINK"):"*) ;;
    *) echo "    Note: $(dirname "$LINK") is not on your PATH; use the menu entry or the full path." ;;
esac

# --- the game needs the SDL2 library from the system ---
if ! { ldconfig -p 2>/dev/null | grep -q 'libSDL2-2\.0\.so\.0'; }; then
    echo
    echo "  SDL2 does not seem to be installed. Install it before playing:"
    echo "    Debian/Ubuntu:  sudo apt install libsdl2-2.0-0"
    echo "    Fedora:         sudo dnf install SDL2"
    echo "    Arch:           sudo pacman -S sdl2-compat"
fi
echo
exit 0
__PAYLOAD__
