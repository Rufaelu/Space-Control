package main

import (
	"encoding/binary"
	"image"
	"image/png"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/veandco/go-sdl2/sdl"
)

const mouseSens = 0.13

func main() {
	runtime.LockOSThread()

	// SC_SHOT=<file.png> renders offscreen for a while, saves one frame and exits.
	// SC_SHOT_T sets how many seconds to simulate, SC_SHOT_MENU=1 captures the title screen.
	shot := os.Getenv("SC_SHOT")

	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		panic(err)
	}
	defer sdl.Quit()
	sdl.GLSetAttribute(sdl.GL_CONTEXT_MAJOR_VERSION, 3)
	sdl.GLSetAttribute(sdl.GL_CONTEXT_MINOR_VERSION, 3)
	sdl.GLSetAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_CORE)
	sdl.GLSetAttribute(sdl.GL_DOUBLEBUFFER, 1)

	flags := uint32(sdl.WINDOW_OPENGL | sdl.WINDOW_MAXIMIZED | sdl.WINDOW_RESIZABLE)
	if shot != "" {
		flags = sdl.WINDOW_OPENGL | sdl.WINDOW_HIDDEN
	}
	window, err := sdl.CreateWindow("SPACE CONTROL", sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_CENTERED, 1280, 720, flags)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()
	glctx, err := window.GLCreateContext()
	if err != nil {
		panic(err)
	}
	defer sdl.GLDeleteContext(glctx)
	if err := gl.Init(); err != nil {
		panic(err)
	}
	sdl.GLSetSwapInterval(1)

	w, h := window.GLGetDrawableSize()
	rend := NewRenderer(w, h, shot != "")
	hud := &HUD{r: rend}
	game := NewGame()
	if shot == "" {
		audio = NewAudio()
	}

	if shot != "" {
		runShot(game, rend, hud, shot)
		return
	}

	fullscreen := false
	last := time.Now()
	for {
		now := time.Now()
		dt := float32(now.Sub(last).Seconds())
		last = now
		if dt > 0.05 {
			dt = 0.05
		}

		start := func() {
			game.Reset()
			sdl.SetRelativeMouseMode(true)
			sfx(sfxUISelect, 1)
		}
		setPaused := func(on bool) {
			if on && game.State == stateRunning {
				game.State, game.MenuSel = statePaused, 0
				sdl.SetRelativeMouseMode(false)
				sfx(sfxUISelect, 1)
			} else if !on && game.State == statePaused {
				game.State = stateRunning
				sdl.SetRelativeMouseMode(true)
				sfx(sfxUISelect, 1)
			}
		}
		setVolume := func(v int) {
			game.Volume = min(max(v, 0), 10)
			if audio != nil {
				audio.Volume = game.Volume
			}
			sfx(sfxUIMove, 1)
		}
		setEngine := func(v int) {
			game.EngineVol = min(max(v, 0), 10)
			sfx(sfxUIMove, 1)
		}
		quit := false
		beginName := func() {
			game.NameBuf = game.Name
			game.State = stateName
			sdl.StartTextInput()
		}
		if game.State == stateName && !sdl.IsTextInputActive() {
			sdl.StartTextInput()
		}
		setDiff := func(d int) {
			game.Diff = (d + diffCount) % diffCount
			sfx(sfxUIMove, 1)
		}
		activate := func(i int) {
			switch i {
			case pauseResume:
				setPaused(false)
			case pauseRestart:
				start()
			case pauseVolume:
				setVolume((game.Volume + 1) % 11)
			case pauseEngine:
				setEngine((game.EngineVol + 1) % 11)
			case pauseMusic:
				game.Music = !game.Music
				sfx(sfxUIMove, 1)
			case pauseTitle:
				game.Reset()
				game.State = stateMenu
				sfx(sfxUISelect, 1)
			case pauseQuit:
				quit = true
			}
		}
		// mouse position in drawable pixels
		mousePx := func(x, y int32) (float32, float32) {
			ww, wh := window.GetSize()
			dw, dh := window.GLGetDrawableSize()
			return float32(x) * float32(dw) / float32(ww), float32(y) * float32(dh) / float32(wh)
		}
		for ev := sdl.PollEvent(); ev != nil; ev = sdl.PollEvent() {
			switch e := ev.(type) {
			case *sdl.QuitEvent:
				return
			case *sdl.MouseMotionEvent:
				game.Look(float32(e.XRel)*mouseSens, float32(e.YRel)*mouseSens)
				if game.State == statePaused {
					x, y := mousePx(e.X, e.Y)
					if i := pauseItemAt(x, y, float32(rend.w), float32(rend.h)); i >= 0 && i != game.MenuSel {
						game.MenuSel = i
						sfx(sfxUIMove, 1)
					}
				}
			case *sdl.MouseButtonEvent:
				if e.Type != sdl.MOUSEBUTTONDOWN {
					break
				}
				if game.State == stateMenu {
					start()
				} else if game.State == statePaused && e.Button == sdl.BUTTON_LEFT {
					x, y := mousePx(e.X, e.Y)
					if i := pauseItemAt(x, y, float32(rend.w), float32(rend.h)); i >= 0 {
						game.MenuSel = i
						activate(i)
					}
				}
			case *sdl.TextInputEvent:
				if game.State == stateName {
					game.NameBuf = strings.TrimLeft(filterName(game.NameBuf+e.GetText()), " ")
				}
			case *sdl.KeyboardEvent:
				if game.State == stateName {
					if e.Type != sdl.KEYDOWN {
						break
					}
					switch e.Keysym.Sym {
					case sdl.K_BACKSPACE:
						if len(game.NameBuf) > 0 {
							game.NameBuf = game.NameBuf[:len(game.NameBuf)-1]
						}
					case sdl.K_RETURN, sdl.K_KP_ENTER:
						if n := cleanName(game.NameBuf); n != "" {
							game.Name = n
							saveName(n)
							game.State = stateMenu
							sdl.StopTextInput()
							sfx(sfxUISelect, 1)
						}
					case sdl.K_ESCAPE:
						if game.Name != "" {
							game.State = stateMenu
							sdl.StopTextInput()
						} else {
							quit = true
						}
					}
					break
				}
				if e.Type != sdl.KEYDOWN || e.Repeat != 0 {
					break
				}
				switch e.Keysym.Sym {
				case sdl.K_ESCAPE, sdl.K_p:
					switch game.State {
					case stateMenu:
						if e.Keysym.Sym == sdl.K_ESCAPE {
							quit = true
						}
					case stateRunning:
						setPaused(true)
					case statePaused:
						setPaused(false)
					case stateGameOver:
						if e.Keysym.Sym == sdl.K_ESCAPE {
							activate(pauseTitle)
						}
					}
				case sdl.K_RETURN, sdl.K_SPACE:
					if game.State == stateMenu {
						start()
					} else if game.State == statePaused {
						activate(game.MenuSel)
					}
				case sdl.K_UP, sdl.K_w:
					if game.State == statePaused {
						game.MenuSel = (game.MenuSel + pauseCount - 1) % pauseCount
						sfx(sfxUIMove, 1)
					}
				case sdl.K_DOWN, sdl.K_s:
					if game.State == statePaused {
						game.MenuSel = (game.MenuSel + 1) % pauseCount
						sfx(sfxUIMove, 1)
					}
				case sdl.K_LEFT, sdl.K_a:
					if game.State == statePaused && game.MenuSel == pauseVolume {
						setVolume(game.Volume - 1)
					} else if game.State == statePaused && game.MenuSel == pauseEngine {
						setEngine(game.EngineVol - 1)
					} else if game.State == stateMenu {
						setDiff(game.Diff - 1)
					}
				case sdl.K_RIGHT, sdl.K_d:
					if game.State == statePaused && game.MenuSel == pauseVolume {
						setVolume(game.Volume + 1)
					} else if game.State == statePaused && game.MenuSel == pauseEngine {
						setEngine(game.EngineVol + 1)
					} else if game.State == stateMenu {
						setDiff(game.Diff + 1)
					}
				case sdl.K_1, sdl.K_2, sdl.K_3:
					if game.State == stateMenu {
						setDiff(int(e.Keysym.Sym - sdl.K_1))
					}
				case sdl.K_m:
					if game.State != stateName {
						game.Music = !game.Music
					}
				case sdl.K_F2:
					if game.State == stateMenu {
						beginName()
					}
				case sdl.K_v, sdl.K_c:
					if game.State == stateRunning {
						game.Cockpit = !game.Cockpit
						sfx(sfxUIMove, 1)
					}
				case sdl.K_r:
					if game.State == stateGameOver {
						start()
					}
				case sdl.K_f, sdl.K_F11:
					fullscreen = !fullscreen
					if fullscreen {
						window.SetFullscreen(sdl.WINDOW_FULLSCREEN_DESKTOP)
					} else {
						window.SetFullscreen(0)
					}
				}
			}
		}

		var in Input
		keys := sdl.GetKeyboardState()
		_, _, mState := sdl.GetMouseState()
		if keys[sdl.SCANCODE_W] != 0 {
			in.Fwd++
		}
		if keys[sdl.SCANCODE_S] != 0 {
			in.Fwd--
		}
		if keys[sdl.SCANCODE_D] != 0 {
			in.Strafe++
		}
		if keys[sdl.SCANCODE_A] != 0 {
			in.Strafe--
		}
		in.Fire = keys[sdl.SCANCODE_SPACE] != 0 || mState&sdl.ButtonLMask() != 0
		in.Boost = keys[sdl.SCANCODE_LSHIFT] != 0 || keys[sdl.SCANCODE_RSHIFT] != 0
		in.FlipL = keys[sdl.SCANCODE_Q] != 0
		in.FlipR = keys[sdl.SCANCODE_E] != 0
		in.Rocket = keys[sdl.SCANCODE_LALT] != 0 || keys[sdl.SCANCODE_RALT] != 0 || mState&sdl.ButtonRMask() != 0

		if quit {
			return
		}
		game.Update(dt, in)
		if audio != nil {
			audio.Listen(game)
			audio.Pump()
		}

		w, h := window.GLGetDrawableSize()
		rend.resize(w, h)
		rend.Draw(game)
		hud.Draw(game)
		window.GLSwap()
	}
}

// runShot plays the game on autopilot offscreen and writes one frame to a PNG.
func runShot(g *Game, r *Renderer, hud *HUD, path string) {
	secs := 14.0
	if v, err := strconv.ParseFloat(os.Getenv("SC_SHOT_T"), 64); err == nil {
		secs = v
	}
	menu := os.Getenv("SC_SHOT_MENU") != ""
	g.State = stateMenu
	if g.Name == "" {
		g.Name = "ACE"
	}
	if v, err := strconv.Atoi(os.Getenv("SC_SHOT_DIFF")); err == nil {
		g.Diff = v
	}
	if os.Getenv("SC_SHOT_NAME") != "" {
		g.State, g.NameBuf, menu = stateName, "RAFA", true
	}
	if !menu {
		g.Reset()
		if v, err := strconv.ParseFloat(os.Getenv("SC_SHOT_YAW"), 32); err == nil {
			g.P.Yaw = float32(v)
		}
		g.waveT = 0.2
		if v, err := strconv.Atoi(os.Getenv("SC_SHOT_WAVE")); err == nil {
			g.Wave = v - 1
		}
	}
	// SC_SHOT_WAV=<file.wav> also records the soundtrack of the run
	var pcm []byte
	wav := os.Getenv("SC_SHOT_WAV")
	if wav != "" {
		audio = newSynth()
	}
	const dt = 1.0 / 60
	for i := 0; i < int(secs*60); i++ {
		in := Input{}
		if !menu {
			// aim at the nearest enemy
			var best *Enemy
			for _, e := range g.Enemies {
				if best == nil || e.Pos.Sub(g.P.Pos).Len() < best.Pos.Sub(g.P.Pos).Len() {
					best = e
				}
			}
			if best != nil && os.Getenv("SC_SHOT_IDLE") == "" {
				f, to := g.P.Front(), norm(best.Pos.Sub(g.P.Pos))
				rt, up := g.P.Right(), g.P.Up()
				g.Look(to.Dot(rt)*4, -to.Dot(up)*4)
				in.Fire = f.Dot(to) > 0.995
				in.Rocket = i%90 == 45
			}
			g.P.Shields = maxShields
		}
		g.Update(dt, in)
		r.Draw(g)
		hud.Draw(g)
		if wav != "" {
			audio.Listen(g)
			pcm = append(pcm, audio.render(sampleRate/60)...)
		}
	}
	if wav != "" {
		hdr := make([]byte, 44)
		copy(hdr, "RIFF")
		binary.LittleEndian.PutUint32(hdr[4:], uint32(36+len(pcm)))
		copy(hdr[8:], "WAVEfmt ")
		binary.LittleEndian.PutUint32(hdr[16:], 16)
		binary.LittleEndian.PutUint16(hdr[20:], 1)
		binary.LittleEndian.PutUint16(hdr[22:], 2)
		binary.LittleEndian.PutUint32(hdr[24:], sampleRate)
		binary.LittleEndian.PutUint32(hdr[28:], sampleRate*4)
		binary.LittleEndian.PutUint16(hdr[32:], 4)
		binary.LittleEndian.PutUint16(hdr[34:], 16)
		copy(hdr[36:], "data")
		binary.LittleEndian.PutUint32(hdr[40:], uint32(len(pcm)))
		if err := os.WriteFile(wav, append(hdr, pcm...), 0o644); err != nil {
			panic(err)
		}
	}
	if os.Getenv("SC_SHOT_COCKPIT") != "" {
		g.Cockpit = true
		r.Draw(g)
		hud.Draw(g)
	}
	if os.Getenv("SC_SHOT_PAUSE") != "" {
		g.State, g.MenuSel = statePaused, 2
		r.Draw(g)
		hud.Draw(g)
	}
	pix := make([]byte, int(r.w)*int(r.h)*4)
	gl.BindFramebuffer(gl.FRAMEBUFFER, r.final.fbo)
	gl.ReadPixels(0, 0, r.w, r.h, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pix))
	img := image.NewNRGBA(image.Rect(0, 0, int(r.w), int(r.h)))
	row := int(r.w) * 4
	for y := 0; y < int(r.h); y++ {
		copy(img.Pix[y*row:(y+1)*row], pix[(int(r.h)-1-y)*row:(int(r.h)-y)*row])
	}
	for i := 3; i < len(img.Pix); i += 4 {
		img.Pix[i] = 255
	}
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}
