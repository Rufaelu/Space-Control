#!/bin/sh
# Rebuilds the setups after you change the game. Run it from anywhere:
#
#   ./build-setup.sh            both setups
#   ./build-setup.sh windows    only the Windows setup
#   ./build-setup.sh linux      only the Linux installer
#
# The results land in dist/ - copy that folder to the other PC:
#   dist/SpaceControl-windows-x86_64-setup.exe
#   dist/SpaceControl-linux-x86_64-installer.run
#   dist/HOW-TO-INSTALL.txt
#
# It needs Docker (it starts Docker Desktop itself if it is not running).
# See installer/build-installers.sh for how it works.
exec sh "$(dirname "$0")/installer/build-installers.sh" "$@"
