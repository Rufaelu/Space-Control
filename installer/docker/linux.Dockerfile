# Build environment for the Linux installer (used by installer/build-installers.sh).
#
# Ubuntu 22.04 has glibc 2.35, so the game built here runs on desktop Linux
# from 2022 on: Ubuntu 22.04+, Debian 12+, Fedora 36+, Mint 21+, Arch...
# SDL2 is built from source with its default run-time loading of X11, Wayland,
# ALSA, PulseAudio and PipeWire, so the libSDL2 shipped beside the game needs
# nothing beyond glibc; distro SDL2 packages hard-link those libraries instead.
FROM ubuntu:22.04

RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        ca-certificates curl gcc g++ make cmake pkg-config binutils \
        libx11-dev libxext-dev libxrandr-dev libxcursor-dev libxfixes-dev libxi-dev \
        libxinerama-dev libxss-dev libxxf86vm-dev libxkbcommon-dev \
        libwayland-dev wayland-protocols libdecor-0-dev libegl1-mesa-dev libgl1-mesa-dev \
        libgles2-mesa-dev libdrm-dev libgbm-dev \
        libasound2-dev libpulse-dev libpipewire-0.3-dev libdbus-1-dev libudev-dev \
    && rm -rf /var/lib/apt/lists/*

# The sed drops SDL's build-time rpath: the game finds libSDL2 beside itself.
ARG SDL_VERSION=2.32.10
RUN curl -fsSL "https://github.com/libsdl-org/SDL/releases/download/release-$SDL_VERSION/SDL2-$SDL_VERSION.tar.gz" \
        | tar -xz -C /tmp \
    && cmake -S "/tmp/SDL2-$SDL_VERSION" -B /tmp/sdl-build -DCMAKE_BUILD_TYPE=Release \
        -DCMAKE_INSTALL_PREFIX=/opt/sdl2 -DCMAKE_INSTALL_LIBDIR=lib \
        -DSDL_STATIC=OFF -DSDL_TEST_LIBRARY=OFF -DSDL_TESTS=OFF \
    && cmake --build /tmp/sdl-build -j"$(nproc)" \
    && cmake --install /tmp/sdl-build --strip \
    && rm -rf "/tmp/SDL2-$SDL_VERSION" /tmp/sdl-build \
    && sed -i 's/-Wl,-rpath,${libdir}//; s/-Wl,--enable-new-dtags//' /opt/sdl2/lib/pkgconfig/sdl2.pc
ENV PKG_CONFIG_PATH=/opt/sdl2/lib/pkgconfig

ARG GO_VERSION=1.26.11
RUN curl -fsSL "https://go.dev/dl/go$GO_VERSION.linux-amd64.tar.gz" | tar -xz -C /usr/local
ENV PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local
