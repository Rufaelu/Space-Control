#!/bin/sh
# Rebuilds the setups after you change the game. Run it from anywhere:
#
#   ./build-setup.sh            all three setups
#   ./build-setup.sh windows    only the Windows setup
#   ./build-setup.sh linux      only the Linux installer
#   ./build-setup.sh android    only the Android app
#
# The results land in dist/ - copy what you need to the other PC or phone:
#   dist/SpaceControl-windows-x86_64-setup.exe
#   dist/SpaceControl-linux-x86_64-installer.run
#   dist/SpaceControl-android.apk
#   dist/HOW-TO-INSTALL.txt
#
# Windows and Linux need Docker (it starts Docker Desktop itself if it is not
# running); Android needs the Android SDK that Android Studio installs.
# See installer/build-installers.sh for how it works.
exec sh "$(dirname "$0")/installer/build-installers.sh" "$@"
