package main

import (
	"strings"

	"github.com/veandco/go-sdl2/sdl"
)

const mouseSens = 0.13

// App connects input (keyboard and mouse, or touch and tilt) to the game.
// Every action lives here once, whichever input triggers it.
type App struct {
	win        *sdl.Window
	rend       *Renderer
	hud        *HUD
	g          *Game
	touch      *Touch // nil without the phone controls
	tilt       *Tilt  // nil without a tilt sensor
	gyro       *Gyro  // nil without a gyroscope
	fullscreen bool
	quit       bool
}

func newApp(win *sdl.Window, rend *Renderer, hud *HUD, g *Game) *App {
	a := &App{win: win, rend: rend, hud: hud, g: g}
	if onPhone {
		a.touch = newTouch()
		up := openTilt()
		a.tilt = newTilt(up)
		a.gyro = newGyro(openGyro(), up)
		openHaptic()
		g.Touch = true
		g.Menu = pauseMenu(true)
		loadControls(g)
		hud.touch = a.touch
	}
	return a
}

// --- actions ---

func (a *App) grabMouse(on bool) {
	if a.touch == nil {
		sdl.SetRelativeMouseMode(on)
	}
}

func (a *App) start() {
	a.g.Reset()
	a.tilt.Recenter() // however the phone is held now counts as level
	a.grabMouse(true)
	sfx(sfxUISelect, 1)
}

func (a *App) setPaused(on bool) {
	g := a.g
	if on && g.State == stateRunning {
		g.State, g.MenuSel = statePaused, 0
		a.grabMouse(false)
		if a.touch != nil {
			a.touch.Reset()
		}
		sfx(sfxUISelect, 1)
	} else if !on && g.State == statePaused {
		g.State = stateRunning
		a.grabMouse(true)
		sfx(sfxUISelect, 1)
	}
}

func (a *App) toTitle() {
	a.g.Reset()
	a.g.State = stateMenu
	a.grabMouse(false) // the title screen has buttons to click
	sfx(sfxUISelect, 1)
}

func (a *App) beginName() {
	a.g.NameBuf = a.g.Name
	a.g.State = stateName
	sdl.StartTextInput()
}

func (a *App) confirmName() {
	if n := cleanName(a.g.NameBuf); n != "" {
		a.g.Name = n
		saveName(n)
		a.g.State = stateMenu
		sdl.StopTextInput()
		sfx(sfxUISelect, 1)
	}
}

func (a *App) setDiff(d int) {
	a.g.Diff = (d + diffCount) % diffCount
	sfx(sfxUIMove, 1)
}

func (a *App) setVolume(v int) {
	a.g.Volume = min(max(v, 0), 10)
	if audio != nil {
		audio.Volume = a.g.Volume
	}
	sfx(sfxUIMove, 1)
}

// back is Esc on a keyboard and the Back button on a phone. It never closes
// the game: that takes QUIT GAME or the window's close button.
func (a *App) back() {
	switch a.g.State {
	case stateRunning:
		a.setPaused(true)
	case statePaused:
		a.setPaused(false)
	case stateGameOver:
		a.toTitle()
	case stateName:
		if a.g.Name != "" {
			a.g.State = stateMenu
			sdl.StopTextInput()
		}
	}
}

// activate runs a pause-menu entry.
func (a *App) activate(item int) {
	g := a.g
	switch item {
	case pauseResume:
		a.setPaused(false)
	case pauseRestart:
		a.start()
	case pauseVolume, pauseEngine, pauseTurn:
		a.adjust(item, 1, true)
	case pauseMusic:
		g.Music = !g.Music
		sfx(sfxUIMove, 1)
	case pauseNextSong:
		if audio != nil {
			audio.NextTrack()
		}
		sfx(sfxUIMove, 1)
	case pauseMotion:
		a.nextMotion()
	case pauseAssist:
		g.Assist = !g.Assist
		saveControls(g)
		sfx(sfxUIMove, 1)
	case pauseAutoFire:
		g.AutoFire = !g.AutoFire
		saveControls(g)
		sfx(sfxUIMove, 1)
	case pauseVibrate:
		g.Vibrate = !g.Vibrate
		saveControls(g)
		sfx(sfxUIMove, 1)
		if g.Vibrate && haptic != nil {
			haptic.RumblePlay(0.5, 60) // so the player feels it is on
		}
	case pauseRecenter:
		a.tilt.Recenter()
		a.setPaused(false)
	case pauseTitle:
		a.toTitle()
	case pauseQuit:
		a.quit = true
	}
}

// nextMotion steps OFF -> TILT -> GYRO, skipping what the phone lacks.
func (a *App) nextMotion() {
	g := a.g
	for range motionNames {
		g.Motion = (g.Motion + 1) % len(motionNames)
		if g.Motion == motionOff || (g.Motion == motionTilt && a.tilt != nil) || (g.Motion == motionGyro && a.gyro != nil) {
			break
		}
	}
	a.tilt.Recenter()
	saveControls(g)
	sfx(sfxUIMove, 1)
}

// adjust steps a level entry; wrap goes from the top back to the bottom.
func (a *App) adjust(item, d int, wrap bool) {
	g := a.g
	step := func(v, lo, hi int) int {
		v += d
		if wrap && v > hi {
			return lo
		}
		return min(max(v, lo), hi)
	}
	switch item {
	case pauseVolume:
		a.setVolume(step(g.Volume, 0, 10))
	case pauseEngine:
		g.EngineVol = step(g.EngineVol, 0, 10)
		sfx(sfxUIMove, 1)
	case pauseTurn:
		g.TurnSpeed = step(g.TurnSpeed, 1, 10)
		saveControls(g)
		sfx(sfxUIMove, 1)
	}
}

func (a *App) selected() int {
	if a.g.MenuSel >= 0 && a.g.MenuSel < len(a.g.Menu) {
		return a.g.Menu[a.g.MenuSel]
	}
	return -1
}

// --- events ---

// px maps a window position to the HUD's pixels.
func (a *App) px(x, y int32) (float32, float32) {
	ww, wh := a.win.GetSize()
	return float32(x) * float32(a.rend.w) / float32(ww), float32(y) * float32(a.rend.h) / float32(wh)
}

func (a *App) handle(ev sdl.Event) {
	g := a.g
	switch e := ev.(type) {
	case *sdl.QuitEvent:
		a.quit = true
	case *sdl.WindowEvent:
		if a.touch == nil {
			return
		}
		switch e.Event {
		case sdl.WINDOWEVENT_FOCUS_LOST: // a call, the home button: pause rather than play on unseen
			a.setPaused(true)
		case sdl.WINDOWEVENT_FOCUS_GAINED: // a dialog may have put the keyboard away
			if g.State == stateName {
				sdl.StartTextInput()
			}
		}
	case *sdl.TouchFingerEvent:
		a.handleTouch(e)
	case *sdl.MouseMotionEvent:
		if a.touch != nil {
			return
		}
		g.Look(float32(e.XRel)*mouseSens, float32(e.YRel)*mouseSens)
		x, y := a.px(e.X, e.Y)
		W, H := float32(a.rend.w), float32(a.rend.h)
		switch g.State {
		case stateMenu:
			q := titleQuitRect(W, H, a.hud.scale(H))
			if hover := q.has(x, y); hover != g.QuitHover {
				g.QuitHover = hover
				if hover {
					sfx(sfxUIMove, 1)
				}
			}
		case statePaused:
			if i := pauseItemAt(x, y, W, H, a.hud.scale(H), len(g.Menu), g.Touch); i >= 0 && i != g.MenuSel {
				g.MenuSel = i
				sfx(sfxUIMove, 1)
			}
		}
	case *sdl.MouseButtonEvent:
		if a.touch != nil || e.Type != sdl.MOUSEBUTTONDOWN {
			return
		}
		x, y := a.px(e.X, e.Y)
		W, H := float32(a.rend.w), float32(a.rend.h)
		switch g.State {
		case stateMenu:
			if titleQuitRect(W, H, a.hud.scale(H)).has(x, y) {
				a.quit = true
			} else {
				a.start()
			}
		case statePaused:
			if e.Button == sdl.BUTTON_LEFT {
				if i := pauseItemAt(x, y, W, H, a.hud.scale(H), len(g.Menu), g.Touch); i >= 0 {
					g.MenuSel = i
					a.activate(g.Menu[i])
				}
			}
		}
	case *sdl.TextInputEvent:
		if g.State == stateName {
			g.NameBuf = strings.TrimLeft(filterName(g.NameBuf+e.GetText()), " ")
		}
	case *sdl.KeyboardEvent:
		a.handleKey(e)
	}
}

func (a *App) handleKey(e *sdl.KeyboardEvent) {
	g := a.g
	if e.Type != sdl.KEYDOWN {
		return
	}
	if g.State == stateName {
		switch e.Keysym.Sym {
		case sdl.K_BACKSPACE:
			if len(g.NameBuf) > 0 {
				g.NameBuf = g.NameBuf[:len(g.NameBuf)-1]
			}
		case sdl.K_RETURN, sdl.K_KP_ENTER:
			a.confirmName()
		case sdl.K_ESCAPE, sdl.K_AC_BACK:
			a.back()
		}
		return
	}
	if e.Repeat != 0 {
		return
	}
	switch e.Keysym.Sym {
	case sdl.K_ESCAPE, sdl.K_AC_BACK:
		a.back()
	case sdl.K_p:
		if g.State == stateRunning || g.State == statePaused {
			a.back()
		}
	case sdl.K_RETURN, sdl.K_SPACE:
		if g.State == stateMenu {
			a.start()
		} else if g.State == statePaused {
			a.activate(a.selected())
		}
	case sdl.K_UP, sdl.K_w:
		if g.State == statePaused {
			g.MenuSel = (g.MenuSel + len(g.Menu) - 1) % len(g.Menu)
			sfx(sfxUIMove, 1)
		}
	case sdl.K_DOWN, sdl.K_s:
		if g.State == statePaused {
			g.MenuSel = (g.MenuSel + 1) % len(g.Menu)
			sfx(sfxUIMove, 1)
		}
	case sdl.K_LEFT, sdl.K_a:
		if g.State == statePaused && adjustable(a.selected()) {
			a.adjust(a.selected(), -1, false)
		} else if g.State == stateMenu {
			a.setDiff(g.Diff - 1)
		}
	case sdl.K_RIGHT, sdl.K_d:
		if g.State == statePaused && adjustable(a.selected()) {
			a.adjust(a.selected(), 1, false)
		} else if g.State == stateMenu {
			a.setDiff(g.Diff + 1)
		}
	case sdl.K_1, sdl.K_2, sdl.K_3:
		if g.State == stateMenu {
			a.setDiff(int(e.Keysym.Sym - sdl.K_1))
		}
	case sdl.K_m:
		g.Music = !g.Music
	case sdl.K_n:
		if audio != nil {
			audio.NextTrack()
		}
	case sdl.K_F2:
		if g.State == stateMenu {
			a.beginName()
		}
	case sdl.K_v, sdl.K_c:
		if g.State == stateRunning {
			g.Cockpit = !g.Cockpit
			sfx(sfxUIMove, 1)
		}
	case sdl.K_r:
		if g.State == stateGameOver {
			a.start()
		}
	case sdl.K_f, sdl.K_F11:
		if a.touch == nil {
			a.fullscreen = !a.fullscreen
			if a.fullscreen {
				a.win.SetFullscreen(sdl.WINDOW_FULLSCREEN_DESKTOP)
			} else {
				a.win.SetFullscreen(0)
			}
		}
	}
}

// handleTouch routes fingers: during play to the controls, otherwise to the
// buttons of whichever screen is up.
func (a *App) handleTouch(e *sdl.TouchFingerEvent) {
	t, g := a.touch, a.g
	if t == nil {
		return
	}
	W, H := float32(a.rend.w), float32(a.rend.h)
	x, y := e.X*W, e.Y*H
	switch e.Type {
	case sdl.FINGERMOTION:
		t.Move(e.FingerID, x, y)
	case sdl.FINGERUP:
		t.Up(e.FingerID)
	case sdl.FINGERDOWN:
		if g.State == stateRunning {
			if t.Down(e.FingerID, x, y) < tbCount {
				g.buzz(0.2, 15) // a tick, so a thumb feels it found the button
			}
			return
		}
		t.Ignore(e.FingerID)
		a.tap(x, y)
	}
}

// tap presses whatever is under a finger on the title, name, pause and
// game-over screens.
func (a *App) tap(x, y float32) {
	g := a.g
	W, H := float32(a.rend.w), float32(a.rend.h)
	s := a.hud.scale(H)
	switch g.State {
	case stateMenu:
		l := titleLayout(W, H, s, true)
		switch {
		case l.quit.has(x, y):
			a.quit = true
		case l.launch.has(x, y):
			a.start()
		case l.name.has(x, y):
			a.beginName()
		default:
			for i, r := range l.levels {
				if r.has(x, y) {
					a.setDiff(i)
				}
			}
		}
	case stateName:
		if nameOKRect(W, H, s).has(x, y) {
			a.confirmName()
		} else {
			sdl.StartTextInput() // bring the keyboard back if it was put away
		}
	case statePaused:
		i := pauseItemAt(x, y, W, H, s, len(g.Menu), true)
		if i < 0 {
			return
		}
		g.MenuSel = i
		item := g.Menu[i]
		if adjustable(item) {
			// the ends of a level entry step it down and up
			r := pauseItemRect(i, W, H, s, len(g.Menu), true)
			switch {
			case x < r.x+r.w*0.3:
				a.adjust(item, -1, false)
			case x > r.x+r.w*0.7:
				a.adjust(item, 1, false)
			}
			return
		}
		a.activate(item)
	case stateGameOver:
		again, title := gameOverButtons(W, H, s)
		if again.has(x, y) {
			a.start()
		} else if title.has(x, y) {
			a.toTitle()
		}
	}
}

// --- per frame ---

// input gathers this frame's controls, and turns the ship by touch and tilt.
func (a *App) input(dt float32) Input {
	var in Input
	g := a.g
	if t := a.touch; t != nil {
		W, H := float32(a.rend.w), float32(a.rend.h)
		t.layout(W, H, a.hud.scale(H))
		if g.State != stateRunning {
			t.Reset()
			return in
		}
		speed := float32(g.TurnSpeed) / 5
		lx, ly := t.Look(dt, speed)
		turning := lx != 0 || ly != 0
		if g.Assist {
			k := g.AssistFriction()
			lx, ly = lx*k, ly*k
		}
		g.Look(lx, ly)
		switch {
		case g.Motion == motionTilt && a.tilt != nil:
			mx, my := a.tilt.Turn(dt, speed)
			g.Look(mx, my)
			turning = turning || mx != 0 || my != 0
		case g.Motion == motionGyro && a.gyro != nil:
			mx, my := a.gyro.Turn(dt, speed)
			if g.Assist {
				k := g.AssistFriction()
				mx, my = mx*k, my*k
			}
			g.Look(mx, my)
			turning = turning || mx != 0 || my != 0
		}
		in.Fwd, in.Strafe = t.StickY, t.StickX
		in.Fire = t.Held[tbFire] || (g.AutoFire && g.AutoFireWant())
		if t.Held[tbTrack] {
			g.TrackTurn(dt, speed)
		} else {
			g.Tracked = nil
			// the pull only helps a player who is already playing
			if g.Assist && (turning || in.Fire || in.Fwd != 0 || in.Strafe != 0) {
				g.AssistPull(dt)
			}
		}
		in.Boost = t.Held[tbBoost] || t.Boosting
		in.Rocket = t.Held[tbRocket]
		in.FlipL = t.Held[tbFlipL]
		in.FlipR = t.Held[tbFlipR]
		if t.Tapped(tbView) {
			g.Cockpit = !g.Cockpit
			sfx(sfxUIMove, 1)
		}
		if t.Tapped(tbPause) {
			a.setPaused(true)
		}
		return in
	}
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
	return in
}
