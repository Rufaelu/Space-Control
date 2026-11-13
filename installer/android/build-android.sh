#!/bin/sh
# Builds dist/SpaceControl-android.apk: Space Control for Android 11 and newer,
# landscape, for 64- and 32-bit ARM phones and the x86_64 emulator.
#
#   sh installer/android/build-android.sh
#
# Needs the Android SDK (with an NDK, build-tools and platform android-35, as
# Android Studio installs them), a JDK, Go and cmake. It does not use Gradle.
# The first run downloads SDL2's source (about 8 MB) and builds it once per
# CPU type, which takes a few minutes; later runs reuse that.
#
# Settings (environment variables):
#   ANDROID_HOME   the SDK (default: ~/Android/Sdk, else /opt/android-sdk)
#   ABIS           CPU types to include (default: arm64-v8a armeabi-v7a x86_64)
#   VERSION_NAME   shown in Android's app info (default: 1.0)
#
# Phones accept an update only if it is signed with the same key, so the APK
# is signed with ~/.android/debug.keystore (made on first use).
set -eu
cd "$(dirname "$0")/../.."
ROOT=$PWD
HERE=$ROOT/installer/android
W=$ROOT/build/android            # work folder (git ignores build/)
SDL_VERSION=2.32.10
MIN_API=30                        # Android 11
TARGET_API=35
ABIS=${ABIS:-arm64-v8a armeabi-v7a x86_64}
VERSION_NAME=${VERSION_NAME:-1.0}
VERSION_CODE=$(date +%s)          # always newer, so a reinstall upgrades
OUT=$ROOT/dist/SpaceControl-android.apk

say() { printf '==> %s\n' "$*"; }
die() { printf 'build-android: %s\n' "$*" >&2; exit 1; }
latest() { ls -d "$@" 2>/dev/null | sort -V | tail -1; }

# --- tools ---
SDK=${ANDROID_HOME:-${ANDROID_SDK_ROOT:-}}
[ -n "$SDK" ] || for d in "$HOME/Android/Sdk" /opt/android-sdk; do [ -d "$d" ] && SDK=$d && break; done
[ -d "${SDK:-}" ] || die "Android SDK not found; set ANDROID_HOME"
NDK=${ANDROID_NDK_HOME:-$(latest "$SDK"/ndk/*)}
[ -d "$NDK" ] || die "no NDK in $SDK/ndk (install it from Android Studio's SDK Manager)"
BT=$(latest "$SDK"/build-tools/*)
[ -x "$BT/aapt2" ] || die "no build-tools in $SDK/build-tools"
ANDROID_JAR=$SDK/platforms/android-$TARGET_API/android.jar
[ -f "$ANDROID_JAR" ] || ANDROID_JAR=$(latest "$SDK"/platforms/android-*)/android.jar
[ -f "$ANDROID_JAR" ] || die "no Android platform in $SDK/platforms"
CMAKE=$(command -v cmake || latest "$SDK"/cmake/*/bin/cmake)
[ -n "$CMAKE" ] || die "cmake not found"
for t in go javac keytool curl; do command -v $t >/dev/null || die "$t not found"; done
TOOLCHAIN=$NDK/toolchains/llvm/prebuilt/linux-x86_64

# --- SDL2 source ---
mkdir -p "$W/deps"
SDL_SRC=$W/deps/SDL2-$SDL_VERSION
if [ ! -d "$SDL_SRC" ]; then
    say "downloading SDL2 $SDL_VERSION source"
    curl -fsSL "https://github.com/libsdl-org/SDL/releases/download/release-$SDL_VERSION/SDL2-$SDL_VERSION.tar.gz" \
        | tar -xz -C "$W/deps"
fi

# --- native code, per CPU type ---
rm -rf "$W/apk" && mkdir -p "$W/apk/lib" "$W/pc"
# go-gl finds OpenGL ES through pkg-config
cat > "$W/pc/egl.pc" <<'EOF'
Name: egl
Description: Android EGL and OpenGL ES 3
Version: 1.4
Libs: -lEGL -lGLESv3
Cflags:
EOF
for abi in $ABIS; do
    case $abi in
        arm64-v8a)   GOARCH=arm64; GOARM=;  TRIPLE=aarch64-linux-android ;;
        armeabi-v7a) GOARCH=arm;   GOARM=7; TRIPLE=armv7a-linux-androideabi ;;
        x86_64)      GOARCH=amd64; GOARM=;  TRIPLE=x86_64-linux-android ;;
        x86)         GOARCH=386;   GOARM=;  TRIPLE=i686-linux-android ;;
        *) die "unknown CPU type $abi" ;;
    esac
    SDL_OUT=$W/sdl/$abi
    if [ ! -f "$SDL_OUT/lib/libSDL2.so" ]; then
        say "$abi: building SDL2 (once)"
        "$CMAKE" -S "$SDL_SRC" -B "$W/sdl-build/$abi" -G Ninja \
            -DCMAKE_TOOLCHAIN_FILE="$NDK/build/cmake/android.toolchain.cmake" \
            -DANDROID_ABI="$abi" -DANDROID_PLATFORM="android-$MIN_API" \
            -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX="$SDL_OUT" -DCMAKE_INSTALL_LIBDIR=lib \
            -DSDL_STATIC=OFF -DSDL_TEST_LIBRARY=OFF -DSDL_TESTS=OFF >"$W/sdl-build-$abi.log" 2>&1 ||
            die "SDL2 configure failed for $abi; see $W/sdl-build-$abi.log"
        "$CMAKE" --build "$W/sdl-build/$abi" >>"$W/sdl-build-$abi.log" 2>&1 &&
            "$CMAKE" --install "$W/sdl-build/$abi" --strip >>"$W/sdl-build-$abi.log" 2>&1 ||
            die "SDL2 build failed for $abi; see $W/sdl-build-$abi.log"
    fi
    say "$abi: building the game"
    mkdir -p "$W/apk/lib/$abi"
    env CGO_ENABLED=1 GOOS=android GOARCH=$GOARCH GOARM=$GOARM \
        CC="$TOOLCHAIN/bin/${TRIPLE}${MIN_API}-clang" \
        PKG_CONFIG_LIBDIR="$SDL_OUT/lib/pkgconfig:$W/pc" PKG_CONFIG_PATH= \
        go build -buildmode=c-shared -tags egl -trimpath \
        -ldflags "-s -w -extldflags=-Wl,-soname,libmain.so" \
        -o "$W/apk/lib/$abi/libmain.so" .
    rm -f "$W/apk/lib/$abi/libmain.h"
    cp "$SDL_OUT/lib/libSDL2.so" "$W/apk/lib/$abi/"
done

# --- Java: SDL's activity and ours, compiled to dex ---
say "compiling the Java side"
rm -rf "$W/classes" && mkdir -p "$W/classes" "$W/dex"
find "$SDL_SRC/android-project/app/src/main/java" "$HERE/java" -name '*.java' >"$W/java-files"
javac -nowarn -Xlint:none --release 11 -encoding UTF-8 -classpath "$ANDROID_JAR" -d "$W/classes" @"$W/java-files" 2>&1 |
    grep -v '^Note:' || true
[ -f "$W/classes/com/spacecontrol/game/SpaceControlActivity.class" ] || die "Java compile failed"
"$BT/d8" --release --min-api $MIN_API --lib "$ANDROID_JAR" --output "$W/dex" $(find "$W/classes" -name '*.class')

# --- resources, manifest, package ---
say "packing the APK"
sed -e "s/@VERSION_CODE@/$VERSION_CODE/" -e "s/@VERSION_NAME@/$VERSION_NAME/" \
    "$HERE/AndroidManifest.xml" >"$W/AndroidManifest.xml"
rm -rf "$W/res.zip" && "$BT/aapt2" compile --dir "$HERE/res" -o "$W/res.zip"
"$BT/aapt2" link -o "$W/unsigned.apk" -I "$ANDROID_JAR" --manifest "$W/AndroidManifest.xml" \
    --min-sdk-version $MIN_API --target-sdk-version $TARGET_API "$W/res.zip"
cp "$W/dex/classes.dex" "$W/apk/"
(cd "$W/apk" && zip -q -r "$W/unsigned.apk" classes.dex lib)
"$BT/zipalign" -f -p 4 "$W/unsigned.apk" "$W/aligned.apk"

KS=$HOME/.android/debug.keystore
if [ ! -f "$KS" ]; then
    mkdir -p "$HOME/.android"
    keytool -genkeypair -keystore "$KS" -storepass android -keypass android -alias androiddebugkey \
        -dname "CN=Android Debug,O=Android,C=US" -keyalg RSA -keysize 2048 -validity 10000 >/dev/null 2>&1
fi
mkdir -p "$ROOT/dist"
"$BT/apksigner" sign --ks "$KS" --ks-pass pass:android --key-pass pass:android \
    --out "$OUT" "$W/aligned.apk"
rm -f "$OUT.idsig"
echo "Built dist/SpaceControl-android.apk ($(du -h "$OUT" | cut -f1)) for: $ABIS"
