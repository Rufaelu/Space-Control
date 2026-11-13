//go:build android

package main

// On Android the game is a shared library (libmain.so) that SDL's Java
// activity loads and starts by calling SDL_main; see installer/android.

/*
#cgo pkg-config: sdl2
#include <stdlib.h>
#include <SDL.h>
static int sdkVersion(void) { return SDL_GetAndroidSDKVersion(); }
static int requestPermission(const char *p) { return SDL_AndroidRequestPermission(p); }
static int displayOrientation(void) { return (int)SDL_GetDisplayOrientation(0); }
*/
import "C"

import (
	"os"
	"path/filepath"
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
)

func main() {} // unused: Android starts the game through SDL_main

//export SDL_main
func SDL_main(argc C.int, argv **C.char) C.int {
	run()
	os.Exit(0) // end the process with the game, so the next launch starts clean
	return 0
}

var onPhone = true

func platformInit() {
	// the name, high score and settings live in the app's private storage
	if p := sdl.AndroidGetInternalStoragePath(); p != "" {
		os.Setenv("XDG_CONFIG_HOME", p)
	}
	sdl.SetHint(sdl.HINT_ORIENTATIONS, "LandscapeLeft LandscapeRight")
	sdl.SetHint(sdl.HINT_TOUCH_MOUSE_EVENTS, "0") // touches are handled as touches
	sdl.SetHint("SDL_MOUSE_TOUCH_EVENTS", "0")
	sdl.SetHint("SDL_ANDROID_TRAP_BACK_BUTTON", "1") // Back pauses instead of closing the app

	// songs come from the phone's Music folder, which needs one permission
	perm := "android.permission.READ_EXTERNAL_STORAGE"
	if C.sdkVersion() >= 33 {
		perm = "android.permission.READ_MEDIA_AUDIO"
	}
	cs := C.CString(perm)
	C.requestPermission(cs)
	C.free(unsafe.Pointer(cs))
}

func setGLAttributes() {
	sdl.GLSetAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_ES)
	sdl.GLSetAttribute(sdl.GL_CONTEXT_MAJOR_VERSION, 3)
	sdl.GLSetAttribute(sdl.GL_CONTEXT_MINOR_VERSION, 0)
	sdl.GLSetAttribute(sdl.GL_DOUBLEBUFFER, 1)
}

func windowFlags(hidden bool) uint32 { return sdl.WINDOW_OPENGL | sdl.WINDOW_FULLSCREEN }

// defaultMusicDir is the phone's own Music folder.
func defaultMusicDir() string {
	root := os.Getenv("EXTERNAL_STORAGE")
	if root == "" {
		root = "/storage/emulated/0"
	}
	return filepath.Join(root, "Music")
}

// toScreen turns a reading in the phone's portrait axes to match the way the
// screen is currently rotated (x right, y up, z out of the screen).
func toScreen(dx, dy, dz float32) (x, y, z float32) {
	switch C.displayOrientation() {
	case C.SDL_ORIENTATION_LANDSCAPE: // portrait's right edge is now the top
		return -dy, dx, dz
	case C.SDL_ORIENTATION_LANDSCAPE_FLIPPED: // portrait's left edge is now the top
		return dy, -dx, dz
	case C.SDL_ORIENTATION_PORTRAIT_FLIPPED:
		return -dx, -dy, dz
	default:
		return dx, dy, dz
	}
}

func openSensor(kind sdl.SensorType) *sdl.Sensor {
	for i := 0; i < sdl.NumSensors(); i++ {
		if sdl.SensorGetDeviceType(i) == kind {
			if s := sdl.SensorOpen(i); s != nil {
				return s
			}
		}
	}
	return nil
}

// accelTilt reads the accelerometer.
type accelTilt struct {
	s   *sdl.Sensor
	buf [3]float32
}

func openTilt() tiltSource {
	if s := openSensor(sdl.SENSOR_ACCEL); s != nil {
		return &accelTilt{s: s}
	}
	return nil
}

// Up returns which way is up in screen terms (x right, y up, z out of the
// screen).
func (t *accelTilt) Up() (x, y, z float32, ok bool) {
	if t.s.GetData(t.buf[:]) != nil {
		return 0, 0, 0, false
	}
	x, y, z = toScreen(t.buf[0], t.buf[1], t.buf[2])
	return x, y, z, true
}

// gyro reads the gyroscope.
type gyro struct {
	s   *sdl.Sensor
	buf [3]float32
}

func openGyro() gyroSource {
	if s := openSensor(sdl.SENSOR_GYRO); s != nil {
		return &gyro{s: s}
	}
	return nil
}

// Rate is the phone's rotation in radians/second, in screen terms.
func (g *gyro) Rate() (x, y, z float32, ok bool) {
	if g.s.GetData(g.buf[:]) != nil {
		return 0, 0, 0, false
	}
	x, y, z = toScreen(g.buf[0], g.buf[1], g.buf[2])
	return x, y, z, true
}
