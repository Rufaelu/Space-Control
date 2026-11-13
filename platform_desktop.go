//go:build !android

package main

import (
	"os"
	"path/filepath"

	"github.com/veandco/go-sdl2/sdl"
)

func main() { run() }

// onPhone is false here; SC_TOUCH=1 shows the phone controls on a desktop anyway (for testing).
var onPhone = os.Getenv("SC_TOUCH") != ""

func platformInit() {}

func setGLAttributes() {
	sdl.GLSetAttribute(sdl.GL_CONTEXT_MAJOR_VERSION, 3)
	sdl.GLSetAttribute(sdl.GL_CONTEXT_MINOR_VERSION, 3)
	sdl.GLSetAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_CORE)
	sdl.GLSetAttribute(sdl.GL_DOUBLEBUFFER, 1)
}

func windowFlags(hidden bool) uint32 {
	if hidden {
		return sdl.WINDOW_OPENGL | sdl.WINDOW_HIDDEN
	}
	return sdl.WINDOW_OPENGL | sdl.WINDOW_MAXIMIZED | sdl.WINDOW_RESIZABLE
}

// defaultMusicDir is the "music" folder next to the game's executable, or
// ./music when running from the source tree.
func defaultMusicDir() string {
	var dirs []string
	if exe, err := os.Executable(); err == nil {
		if real, err := filepath.EvalSymlinks(exe); err == nil {
			exe = real
		}
		dirs = append(dirs, filepath.Join(filepath.Dir(exe), "music"))
	}
	if wd, err := os.Getwd(); err == nil {
		dirs = append(dirs, filepath.Join(wd, "music"))
	}
	for _, d := range dirs {
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			return d
		}
	}
	if len(dirs) > 0 {
		return dirs[0] // not there yet; picked up if it is created later
	}
	return "music"
}

// openTilt and openGyro find no motion sensors on a desktop.
func openTilt() tiltSource { return nil }
func openGyro() gyroSource { return nil }
