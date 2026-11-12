# Build environment for the Windows installer (used by installer/build-installers.sh):
# cross-compiles from Linux with MinGW-w64, links the official SDL2 MinGW
# development release, and packs the setup program with NSIS.
FROM debian:bookworm

RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates curl gcc-mingw-w64-x86-64 binutils-mingw-w64-x86-64 nsis \
    && rm -rf /var/lib/apt/lists/*

ARG SDL_VERSION=2.32.10
RUN mkdir -p /opt/sdl2-mingw \
    && curl -fsSL "https://github.com/libsdl-org/SDL/releases/download/release-$SDL_VERSION/SDL2-devel-$SDL_VERSION-mingw.tar.gz" \
        | tar -xz -C /opt/sdl2-mingw --strip-components=1
ENV SDL2_MINGW=/opt/sdl2-mingw/x86_64-w64-mingw32

ARG GO_VERSION=1.26.11
RUN curl -fsSL "https://go.dev/dl/go$GO_VERSION.linux-amd64.tar.gz" | tar -xz -C /usr/local
ENV PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local
