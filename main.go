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

	"game/internal/gl"
	"github.com/veandco/go-sdl2/sdl"
)

// run is the whole game; main (desktop) or SDL_main (Android) calls it.
func run() {
	runtime.LockOSThread()

	// SC_SHOT=<file.png> renders offscreen for a while, saves one frame and exits.
	// SC_SHOT_T sets how many seconds to simulate, SC_SHOT_MENU=1 captures the title screen.
	shot := os.Getenv("SC_SHOT")

	platformInit()
	if err := sdl.Init(sdl.INIT_VIDEO | sdl.INIT_SENSOR); err != nil {
		panic(err)
	}
	defer sdl.Quit()
	setGLAttributes()

	ww, wh := int32(1280), int32(720)
	if sz := strings.Split(os.Getenv("SC_SHOT_SIZE"), "x"); shot != "" && len(sz) == 2 { // e.g. 1600x720, a phone's shape
		if a, err := strconv.Atoi(sz[0]); err == nil {
			ww = int32(a)
		}
		if b, err := strconv.Atoi(sz[1]); err == nil {
			wh = int32(b)
		}
	}
	window, err := sdl.CreateWindow("SPACE CONTROL", sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_CENTERED, ww, wh, windowFlags(shot != ""))
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
	app := newApp(window, rend, hud, game)
	if shot == "" {
		audio = NewAudio()
	}

	if shot != "" {
		runShot(app, shot)
		return
	}

	last := time.Now()
	for !app.quit {
		now := time.Now()
		dt := float32(now.Sub(last).Seconds())
		last = now
		if dt > 0.05 {
			dt = 0.05
		}
		if game.State == stateName && !sdl.IsTextInputActive() {
			sdl.StartTextInput()
		}
		for ev := sdl.PollEvent(); ev != nil; ev = sdl.PollEvent() {
			app.handle(ev)
		}
		if app.quit {
			return
		}
		in := app.input(dt)
		game.Update(dt, in)
		if audio != nil {
			audio.Listen(game)
			audio.Pump()
			game.SetTrack(audio.Track())
		}

		w, h := window.GLGetDrawableSize()
		rend.resize(w, h)
		rend.Draw(game)
		hud.Draw(game)
		window.GLSwap()
	}
}

// runShot plays the game on autopilot offscreen and writes one frame to a PNG.
func runShot(app *App, path string) {
	g, r, hud := app.g, app.rend, app.hud
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
		audio.music.Block = true // render the songs in step, however fast the frames go
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
		if t := app.touch; t != nil {
			t.layout(float32(r.w), float32(r.h), hud.scale(float32(r.h)))
			// SC_SHOT_PRESS=1 shows a thumb on the stick and on FIRE; =boost pushes
			// the stick into its boost zone. SC_SHOT_TRACK=1 holds TRACK for the last second.
			W, H, hs := float32(r.w), float32(r.h), hud.scale(float32(r.h))
			if press := os.Getenv("SC_SHOT_PRESS"); press != "" && i == int(secs*60)-1 {
				t.Down(1, W*0.12, H*0.72)
				if press == "boost" {
					t.Move(1, W*0.125, H*0.72-stickRadius*hs*1.5)
				} else {
					t.Move(1, W*0.14, H*0.62)
				}
				fx, fy, _ := touchButton(tbFire, W, H, hs)
				t.Down(2, fx, fy)
			}
			if os.Getenv("SC_SHOT_TRACK") != "" && i >= int(secs*60)-60 {
				if i == int(secs*60)-60 {
					tx, ty, _ := touchButton(tbTrack, W, H, hs)
					t.Down(3, tx, ty)
				}
				g.TrackTurn(dt, 1)
			}
		}
		r.Draw(g)
		hud.Draw(g)
		if wav != "" {
			audio.Listen(g)
			pcm = append(pcm, audio.render(sampleRate/60)...)
			g.SetTrack(audio.Track())
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
	if os.Getenv("SC_SHOT_OVER") != "" {
		g.State = stateGameOver
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
