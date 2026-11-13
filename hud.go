package main

import (
	"fmt"
	"math"

	"game/internal/gl"
)

type RGBA = [4]float32

var (
	cCyan  = RGBA{0.40, 0.92, 1.0, 1}
	cWhite = RGBA{0.92, 0.96, 1.0, 1}
	cDim   = RGBA{0.40, 0.92, 1.0, 0.35}
	cRed   = RGBA{1.0, 0.28, 0.22, 1}
	cAmber = RGBA{1.0, 0.70, 0.20, 1}
	cGreen = RGBA{0.35, 1.0, 0.50, 1}
)

func fade(c RGBA, a float32) RGBA { c[3] *= a; return c }

// 5x7 pixel font.
var glyphs = map[rune][7]string{
	'A': {".###.", "#...#", "#...#", "#####", "#...#", "#...#", "#...#"},
	'B': {"####.", "#...#", "#...#", "####.", "#...#", "#...#", "####."},
	'C': {".###.", "#...#", "#....", "#....", "#....", "#...#", ".###."},
	'D': {"####.", "#...#", "#...#", "#...#", "#...#", "#...#", "####."},
	'E': {"#####", "#....", "#....", "####.", "#....", "#....", "#####"},
	'F': {"#####", "#....", "#....", "####.", "#....", "#....", "#...."},
	'G': {".###.", "#...#", "#....", "#.###", "#...#", "#...#", ".###."},
	'H': {"#...#", "#...#", "#...#", "#####", "#...#", "#...#", "#...#"},
	'I': {"#####", "..#..", "..#..", "..#..", "..#..", "..#..", "#####"},
	'J': {"..###", "...#.", "...#.", "...#.", "...#.", "#..#.", ".##.."},
	'K': {"#...#", "#..#.", "#.#..", "##...", "#.#..", "#..#.", "#...#"},
	'L': {"#....", "#....", "#....", "#....", "#....", "#....", "#####"},
	'M': {"#...#", "##.##", "#.#.#", "#.#.#", "#...#", "#...#", "#...#"},
	'N': {"#...#", "##..#", "#.#.#", "#..##", "#...#", "#...#", "#...#"},
	'O': {".###.", "#...#", "#...#", "#...#", "#...#", "#...#", ".###."},
	'P': {"####.", "#...#", "#...#", "####.", "#....", "#....", "#...."},
	'Q': {".###.", "#...#", "#...#", "#...#", "#.#.#", "#..#.", ".##.#"},
	'R': {"####.", "#...#", "#...#", "####.", "#.#..", "#..#.", "#...#"},
	'S': {".####", "#....", "#....", ".###.", "....#", "....#", "####."},
	'T': {"#####", "..#..", "..#..", "..#..", "..#..", "..#..", "..#.."},
	'U': {"#...#", "#...#", "#...#", "#...#", "#...#", "#...#", ".###."},
	'V': {"#...#", "#...#", "#...#", "#...#", "#...#", ".#.#.", "..#.."},
	'W': {"#...#", "#...#", "#...#", "#.#.#", "#.#.#", "##.##", "#...#"},
	'X': {"#...#", "#...#", ".#.#.", "..#..", ".#.#.", "#...#", "#...#"},
	'Y': {"#...#", "#...#", ".#.#.", "..#..", "..#..", "..#..", "..#.."},
	'Z': {"#####", "....#", "...#.", "..#..", ".#...", "#....", "#####"},
	'0': {".###.", "#...#", "#..##", "#.#.#", "##..#", "#...#", ".###."},
	'1': {"..#..", ".##..", "..#..", "..#..", "..#..", "..#..", ".###."},
	'2': {".###.", "#...#", "....#", "...#.", "..#..", ".#...", "#####"},
	'3': {"####.", "....#", "....#", ".###.", "....#", "....#", "####."},
	'4': {"...#.", "..##.", ".#.#.", "#..#.", "#####", "...#.", "...#."},
	'5': {"#####", "#....", "####.", "....#", "....#", "#...#", ".###."},
	'6': {".###.", "#....", "#....", "####.", "#...#", "#...#", ".###."},
	'7': {"#####", "....#", "...#.", "..#..", "..#..", "..#..", "..#.."},
	'8': {".###.", "#...#", "#...#", ".###.", "#...#", "#...#", ".###."},
	'9': {".###.", "#...#", "#...#", ".####", "....#", "....#", ".###."},
	':': {".....", "..#..", ".....", ".....", ".....", "..#..", "....."},
	'.': {".....", ".....", ".....", ".....", ".....", ".....", "..#.."},
	'-': {".....", ".....", ".....", ".###.", ".....", ".....", "....."},
	'!': {"..#..", "..#..", "..#..", "..#..", "..#..", ".....", "..#.."},
	'/': {"....#", "....#", "...#.", "..#..", ".#...", "#....", "#...."},
	'+': {".....", "..#..", "..#..", "#####", "..#..", "..#..", "....."},
	',': {".....", ".....", ".....", ".....", ".....", "..#..", ".#..."},
	'?': {".###.", "#...#", "....#", "...#.", "..#..", ".....", "..#.."},
	'_': {".....", ".....", ".....", ".....", ".....", ".....", "#####"},
}

type HUD struct {
	r     *Renderer
	s     float32 // UI scale
	touch *Touch  // phone controls to draw, or nil
}

func (h *HUD) vert(x, y float32, c RGBA) {
	h.r.hudB.data = append(h.r.hudB.data, x, y, c[0], c[1], c[2], c[3])
}

func (h *HUD) tri(x0, y0, x1, y1, x2, y2 float32, c RGBA) {
	h.vert(x0, y0, c)
	h.vert(x1, y1, c)
	h.vert(x2, y2, c)
}

func (h *HUD) rect(x, y, w, hh float32, c RGBA) {
	h.tri(x, y, x+w, y, x+w, y+hh, c)
	h.tri(x, y, x+w, y+hh, x, y+hh, c)
}

func (h *HUD) line(x0, y0, x1, y1, th float32, c RGBA) {
	dx, dy := x1-x0, y1-y0
	l := float32(math.Hypot(float64(dx), float64(dy)))
	if l < 1e-3 {
		return
	}
	nx, ny := -dy/l*th/2, dx/l*th/2
	h.tri(x0+nx, y0+ny, x1+nx, y1+ny, x1-nx, y1-ny, c)
	h.tri(x0+nx, y0+ny, x1-nx, y1-ny, x0-nx, y0-ny, c)
}

func (h *HUD) arc(cx, cy, rad, th float32, a0, a1 float64, c RGBA) {
	const seg = 48
	n := int(math.Ceil(math.Abs(a1-a0) / (2 * math.Pi) * seg))
	if n < 2 {
		n = 2
	}
	for i := 0; i < n; i++ {
		t0 := a0 + (a1-a0)*float64(i)/float64(n)
		t1 := a0 + (a1-a0)*float64(i+1)/float64(n)
		c0, s0 := float32(math.Cos(t0)), float32(math.Sin(t0))
		c1, s1 := float32(math.Cos(t1)), float32(math.Sin(t1))
		ri, ro := rad-th/2, rad+th/2
		h.tri(cx+c0*ri, cy+s0*ri, cx+c0*ro, cy+s0*ro, cx+c1*ro, cy+s1*ro, c)
		h.tri(cx+c0*ri, cy+s0*ri, cx+c1*ro, cy+s1*ro, cx+c1*ri, cy+s1*ri, c)
	}
}

func (h *HUD) disc(cx, cy, rad float32, c RGBA) { h.arc(cx, cy, rad/2, rad, 0, 2*math.Pi, c) }

func textW(s string, px float32) float32 { return float32(len(s))*6*px - px }

// text draws at top-left (x,y); align: 0 left, 1 centre, 2 right.
func (h *HUD) text(x, y, px float32, s string, c RGBA, align int) {
	switch align {
	case 1:
		x -= textW(s, px) / 2
	case 2:
		x -= textW(s, px)
	}
	for _, ch := range s {
		if g, ok := glyphs[ch]; ok {
			for row := 0; row < 7; row++ {
				for col := 0; col < 5; {
					if g[row][col] != '#' {
						col++
						continue
					}
					run := col
					for run < 5 && g[row][run] == '#' {
						run++
					}
					h.rect(x+float32(col)*px, y+float32(row)*px, float32(run-col)*px, px, c)
					col = run
				}
			}
		}
		x += 6 * px
	}
}

// glowText draws text with a soft halo behind it.
func (h *HUD) glowText(x, y, px float32, s string, c RGBA, align int) {
	halo := fade(c, 0.16)
	for _, o := range [][2]float32{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		h.text(x+o[0]*px*0.6, y+o[1]*px*0.6, px, s, halo, align)
	}
	h.text(x, y, px, s, c, align)
}

func (h *HUD) bar(x, y, w, hh, frac float32, c RGBA, segs int) {
	gap := 2 * h.s
	sw := (w - gap*float32(segs-1)) / float32(segs)
	for i := 0; i < segs; i++ {
		col := fade(c, 0.16)
		if float32(i) < frac*float32(segs)-0.001 {
			col = c
		}
		h.rect(x+float32(i)*(sw+gap), y, sw, hh, col)
	}
}

func (h *HUD) Draw(g *Game) {
	r := h.r
	W, H := float32(r.w), float32(r.h)
	h.s = h.scale(H)
	blinkOn := int(g.Time*6)%2 == 0

	switch g.State {
	case stateMenu:
		if h.touch != nil {
			h.drawTitleTouch(g, W, H, blinkOn)
		} else {
			h.drawTitle(g, W, H, blinkOn)
		}
	case stateName:
		h.drawName(g, W, H, blinkOn)
	default:
		h.drawPlay(g, W, H, blinkOn)
	}
	if g.State == statePaused {
		h.drawPause(g, W, H)
	}
	if g.State == stateGameOver {
		h.drawGameOver(g, W, H, blinkOn)
	}

	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	r.hud.use()
	r.hud.f2("uRes", W, H)
	r.hudB.flush()
	gl.Disable(gl.BLEND)
}

// scale is the HUD's unit: the HUD is laid out for a 720-pixel-tall screen,
// and drawn larger with the phone controls.
func (h *HUD) scale(H float32) float32 {
	if h.touch != nil {
		return H / 720 * touchUI
	}
	return H / 720
}

// rect is a box on screen, in HUD pixels.
type rect struct{ x, y, w, h float32 }

func (r rect) has(px, py float32) bool {
	return px >= r.x && px <= r.x+r.w && py >= r.y && py <= r.y+r.h
}

// button draws a box with a centred label, lit when on.
func (h *HUD) button(r rect, label string, px float32, c RGBA, on bool) {
	s := h.s
	if on {
		h.rect(r.x, r.y, r.w, r.h, fade(c, 0.30))
		h.rect(r.x, r.y+r.h-3*s, r.w, 3*s, c)
		h.glowText(r.x+r.w/2, r.y+r.h/2-3.5*px, px, label, cWhite, 1)
		return
	}
	h.rect(r.x, r.y, r.w, r.h, RGBA{0, 0, 0, 0.45})
	h.rect(r.x, r.y+r.h-2*s, r.w, 2*s, fade(c, 0.6))
	h.text(r.x+r.w/2, r.y+r.h/2-3.5*px, px, label, fade(c, 0.9), 1)
}

func (h *HUD) drawTitle(g *Game, W, H float32, blinkOn bool) {
	s := h.s
	h.glowText(W/2, H*0.16, 11*s, "SPACE CONTROL", cCyan, 1)
	h.text(W/2, H*0.16+92*s, 2.5*s, "HOLD THE LINE. WATCH YOUR HEAT.", cWhite, 1)
	if g.Name != "" {
		h.text(W/2, H*0.16+120*s, 2.2*s, "PILOT  "+g.Name, cAmber, 1)
	}
	// difficulty level picker
	ly := H * 0.645
	h.rect(W/2-280*s, ly-8*s, 560*s, 66*s, RGBA{0, 0, 0, 0.45})
	h.text(W/2, ly, 1.8*s, "LEVEL", fade(cCyan, 0.8), 1)
	for i, d := range diffs {
		x := W/2 + float32(i-1)*170*s
		if i == g.Diff {
			h.rect(x-70*s, ly+18*s, 140*s, 32*s, fade(cCyan, 0.18))
			h.rect(x-70*s, ly+48*s, 140*s, 2*s, cCyan)
			h.glowText(x, ly+24*s, 3*s, d.name, cWhite, 1)
		} else {
			h.text(x, ly+24*s, 3*s, d.name, fade(cCyan, 0.6), 1)
		}
	}
	if blinkOn {
		h.glowText(W/2, H*0.755, 3.2*s, "CLICK OR PRESS ENTER TO LAUNCH", cAmber, 1)
	}
	lines := []string{
		"MOUSE - AIM      W A S D - THRUST",
		"SPACE / LEFT CLICK - FIRE      SHIFT - BOOST",
		"Q - FLIP LEFT      E - FLIP RIGHT      ALT / RIGHT CLICK - ROCKET      V - COCKPIT VIEW",
		"A / D - CHANGE LEVEL      F2 - CHANGE NAME",
		"ESC / P - PAUSE MENU      M - MUSIC      N - NEXT SONG      F - FULLSCREEN",
	}
	for i, l := range lines {
		h.text(W/2, H*0.805+float32(i)*20*s, 1.8*s, l, fade(cCyan, 0.8), 1)
	}
	if g.HiScore > 0 {
		h.text(W/2, H*0.16+144*s, 2*s, fmt.Sprintf("BEST %06d", g.HiScore), cWhite, 1)
	}
	// the only way out from here, besides the window's own close button
	h.button(titleQuitRect(W, H, s), "QUIT GAME", 2.2*s, cRed, g.QuitHover)
	if g.Track != "" {
		h.text(28*s, H-34*s, 1.8*s, "NOW PLAYING  "+g.Track, fade(cAmber, 0.8), 0)
	}
}

// titleQuitRect is the QUIT GAME button in the title screen's top-right corner.
func titleQuitRect(W, H, s float32) rect {
	w, hh := 160*s, 38*s
	return rect{W - w - 24*s, 24 * s, w, hh}
}

// titleTouch is what can be tapped on the phone title screen.
type titleTouch struct {
	quit, launch, name rect
	levels             [diffCount]rect
}

func titleLayout(W, H, s float32, touch bool) titleTouch {
	l := titleTouch{quit: titleQuitRect(W, H, s)}
	l.name = rect{W/2 - 260*s, H*0.09 + 96*s, 520 * s, 30 * s}
	ly := H*0.47 + 20*s
	for i := range l.levels {
		x := W/2 + float32(i-1)*190*s
		l.levels[i] = rect{x - 85*s, ly, 170 * s, 46 * s}
	}
	l.launch = rect{W/2 - 180*s, H * 0.70, 360 * s, 64 * s}
	return l
}

func (h *HUD) drawTitleTouch(g *Game, W, H float32, blinkOn bool) {
	s := h.s
	l := titleLayout(W, H, s, true)
	h.glowText(W/2, H*0.09, 9*s, "SPACE CONTROL", cCyan, 1)
	h.text(W/2, H*0.09+72*s, 2.2*s, "HOLD THE LINE. WATCH YOUR HEAT.", cWhite, 1)
	name := "TAP TO SET YOUR PILOT NAME"
	if g.Name != "" {
		name = "PILOT  " + g.Name + "    TAP TO CHANGE"
	}
	h.text(W/2, l.name.y+l.name.h/2-8*s, 2.4*s, name, cAmber, 1)
	if g.HiScore > 0 {
		h.text(W/2, l.name.y+l.name.h+8*s, 2*s, fmt.Sprintf("BEST %06d", g.HiScore), cWhite, 1)
	}
	h.text(W/2, l.levels[0].y-20*s, 2*s, "LEVEL", fade(cCyan, 0.8), 1)
	for i, d := range diffs {
		h.button(l.levels[i], d.name, 3*s, cCyan, i == g.Diff)
	}
	h.button(l.launch, "LAUNCH", 4.2*s, cAmber, blinkOn)
	h.text(W/2, l.launch.y+l.launch.h+16*s, 1.8*s, "LEFT STICK - FLY      DRAG OR TILT - TURN", fade(cCyan, 0.8), 1)
	h.button(l.quit, "QUIT GAME", 2.2*s, cRed, false)
	if g.Track != "" {
		h.text(28*s, H-34*s, 1.8*s, "NOW PLAYING  "+g.Track, fade(cAmber, 0.8), 0)
	}
}

// nameOKRect is the OK button on the phone's name screen, beside the name:
// in landscape the on-screen keyboard covers all but the top third.
func nameOKRect(W, H, s float32) rect { return rect{W/2 + 230*s, H*0.13 - 4*s, 160 * s, 56 * s} }

func (h *HUD) drawName(g *Game, W, H float32, blinkOn bool) {
	s := h.s
	typed := g.NameBuf
	if blinkOn && len(typed) < maxNameLen {
		typed += "_"
	}
	if h.touch != nil {
		h.rect(0, 0, W, H*0.30, RGBA{0, 0, 0, 0.55})
		h.text(W/2, H*0.04, 3*s, "WHAT IS YOUR NAME, PILOT?", cWhite, 1)
		h.glowText(W/2-textW(g.NameBuf, 6*s)/2, H*0.13, 6*s, typed, cAmber, 0)
		h.button(nameOKRect(W, H, s), "OK", 3.4*s, cCyan, cleanName(g.NameBuf) != "")
		return
	}
	h.glowText(W/2, H*0.16, 11*s, "SPACE CONTROL", cCyan, 1)
	h.rect(0, H*0.40, W, H*0.30, RGBA{0, 0, 0, 0.5})
	h.text(W/2, H*0.44, 3*s, "WHAT IS YOUR NAME, PILOT?", cWhite, 1)
	h.glowText(W/2-textW(g.NameBuf, 6*s)/2, H*0.52, 6*s, typed, cAmber, 0)
	h.text(W/2, H*0.63, 2*s, "TYPE YOUR NAME      ENTER - CONFIRM", fade(cCyan, 0.8), 1)
}

func pauseLabel(g *Game, item int) string {
	onOff := func(b bool) string {
		if b {
			return "ON"
		}
		return "OFF"
	}
	switch item {
	case pauseResume:
		return "RESUME"
	case pauseRestart:
		return "RESTART"
	case pauseVolume:
		return fmt.Sprintf("VOLUME  %d/10", g.Volume)
	case pauseEngine:
		return fmt.Sprintf("ENGINES  %d/10", g.EngineVol)
	case pauseMusic:
		return "MUSIC  " + onOff(g.Music)
	case pauseNextSong:
		return "NEXT SONG"
	case pauseMotion:
		return "MOTION AIM  " + motionNames[g.Motion]
	case pauseTurn:
		return fmt.Sprintf("TURN SPEED  %d/10", g.TurnSpeed)
	case pauseRecenter:
		return "RECENTER TILT"
	case pauseAssist:
		return "AIM ASSIST  " + onOff(g.Assist)
	case pauseAutoFire:
		return "AUTO FIRE  " + onOff(g.AutoFire)
	case pauseVibrate:
		return "VIBRATION  " + onOff(g.Vibrate)
	case pauseTitle:
		return "QUIT TO TITLE"
	default:
		return "QUIT GAME"
	}
}

func (h *HUD) drawPause(g *Game, W, H float32) {
	s := h.s
	touch := h.touch != nil
	if touch {
		h.glowText(W/2, H*0.07, 6*s, "PAUSED", cWhite, 1)
	} else {
		h.glowText(W/2, H*0.20, 8*s, "PAUSED", cWhite, 1)
	}
	px := 3 * s
	if touch {
		px = 2.4 * s
	}
	for i, item := range g.Menu {
		r := pauseItemRect(i, W, H, s, len(g.Menu), touch)
		l := pauseLabel(g, item)
		ty := r.y + r.h/2 - 3.5*px
		if adjustable(item) {
			ty -= 3 * s // room for the level pips
		}
		if i == g.MenuSel && !touch {
			h.rect(r.x, r.y, r.w, r.h, fade(cCyan, 0.18))
			h.rect(r.x, r.y, 4*s, r.h, cCyan)
			h.rect(r.x+r.w-4*s, r.y, 4*s, r.h, cCyan)
			h.glowText(r.x+r.w/2, ty, px, l, cWhite, 1)
		} else {
			if touch {
				h.rect(r.x, r.y, r.w, r.h, RGBA{0, 0, 0, 0.45})
				h.rect(r.x, r.y+r.h-2*s, r.w, 2*s, fade(cCyan, 0.5))
			}
			h.text(r.x+r.w/2, ty, px, l, fade(cCyan, 0.85), 1)
		}
		if adjustable(item) {
			level, hi := g.Volume, 10
			switch item {
			case pauseEngine:
				level = g.EngineVol
			case pauseTurn:
				level = g.TurnSpeed
			}
			pw := 14 * s
			for k := 0; k < hi; k++ {
				c := fade(cCyan, 0.2)
				if k < level {
					c = cAmber
				}
				h.rect(r.x+r.w/2-5*pw+float32(k)*pw+1*s, r.y+r.h-9*s, pw-2*s, 3*s, c)
			}
			if touch { // tap the ends to step down and up
				h.text(r.x+16*s, r.y+r.h/2-3.5*3.2*s, 3.2*s, "-", cAmber, 0)
				h.text(r.x+r.w-16*s, r.y+r.h/2-3.5*3.2*s, 3.2*s, "+", cAmber, 2)
			}
		}
	}
	if !touch {
		h.text(W/2, H*0.80, 1.8*s, "W / S OR MOUSE - CHOOSE      ENTER OR CLICK - SELECT      A / D - ADJUST      ESC - RESUME", fade(cCyan, 0.7), 1)
	}
}

// pauseItemRect is the on-screen box of pause-menu entry i of n. With the
// phone controls the entries are larger and split into two columns.
func pauseItemRect(i int, W, H, s float32, n int, touch bool) rect {
	if !touch {
		w, hh := 380*s, 40*s
		return rect{W/2 - w/2, H*0.31 + float32(i)*46*s, w, hh}
	}
	rows := (n + 1) / 2
	col, row := i/rows, i%rows
	w, hh := 340*s, 46*s
	return rect{W/2 - 355*s + float32(col)*370*s, H*0.21 + float32(row)*56*s, w, hh}
}

// pauseItemAt returns the pause-menu entry under a pixel, or -1.
func pauseItemAt(px, py, W, H, s float32, n int, touch bool) int {
	for i := 0; i < n; i++ {
		if pauseItemRect(i, W, H, s, n, touch).has(px, py) {
			return i
		}
	}
	return -1
}

// gameOverButtons are FLY AGAIN and TITLE on the phone's game-over screen.
func gameOverButtons(W, H, s float32) (again, title rect) {
	w, hh := 250*s, 60*s
	y := H*0.30 + H*0.36 + 18*s
	return rect{W/2 - w - 20*s, y, w, hh}, rect{W/2 + 20*s, y, w, hh}
}

func (h *HUD) drawGameOver(g *Game, W, H float32, blinkOn bool) {
	s := h.s
	h.rect(0, H*0.30, W, H*0.36, RGBA{0, 0, 0, 0.45})
	h.glowText(W/2, H*0.34, 8*s, "SHIP DESTROYED", cRed, 1)
	h.text(W/2, H*0.34+78*s, 3.4*s, fmt.Sprintf("SCORE %06d", g.Score), cWhite, 1)
	h.text(W/2, H*0.34+112*s, 2.4*s, fmt.Sprintf("WAVE %d      KILLS %d", g.Wave, g.Kills), cCyan, 1)
	if g.NewHi {
		h.glowText(W/2, H*0.34+142*s, 2.8*s, "NEW BEST!", cAmber, 1)
	} else {
		h.text(W/2, H*0.34+142*s, 2.2*s, fmt.Sprintf("BEST %06d", g.HiScore), fade(cWhite, 0.7), 1)
	}
	if h.touch != nil {
		again, title := gameOverButtons(W, H, s)
		h.button(again, "FLY AGAIN", 3*s, cAmber, true)
		h.button(title, "TITLE", 3*s, cCyan, false)
		return
	}
	if blinkOn {
		h.text(W/2, H*0.34+182*s, 2.6*s, "R - FLY AGAIN      ESC - TITLE", cAmber, 1)
	}
}

func (h *HUD) drawPlay(g *Game, W, H float32, blinkOn bool) {
	r, s, p := h.r, h.s, &g.P
	m := 28 * s

	// cockpit frame: canopy struts and a dashboard the instruments sit on
	if r.cockpit {
		dark := RGBA{0.025, 0.04, 0.065, 0.94}
		edge := fade(cCyan, 0.55)
		lx, rx, ty, sy := W*0.22, W*0.78, H*0.765, H*0.885
		h.tri(0, sy, lx, ty, 0, H, dark)
		h.tri(lx, ty, lx, H, 0, H, dark)
		h.rect(lx, ty, rx-lx, H-ty, dark)
		h.tri(rx, ty, W, sy, W, H, dark)
		h.tri(rx, ty, W, H, rx, H, dark)
		for _, sx := range []float32{-1, 1} {
			bx, tx := W/2+sx*W*0.28, W/2+sx*W*0.13
			h.line(bx, ty, tx, -10*s, 16*s, dark)
			h.line(bx-sx*8*s, ty, tx-sx*8*s, -10*s, 1.5*s, edge)
			h.line(W/2+sx*W*0.5, sy, bx, ty, 2*s, edge)
		}
		h.line(lx, ty, rx, ty, 2*s, edge)
		if p.Invuln > 0 && blinkOn {
			h.line(lx, ty, rx, ty, 4*s, cRed)
		}
	}

	// score + combo
	h.text(m, m, 2*s, "SCORE", fade(cCyan, 0.8), 0)
	h.glowText(m, m+18*s, 4.2*s, fmt.Sprintf("%06d", g.Score), cWhite, 0)
	h.text(m, m+56*s, 1.8*s, fmt.Sprintf("BEST %06d", max(g.HiScore, g.Score)), fade(cCyan, 0.6), 0)
	if g.Combo > 1 && g.ComboT > 0 {
		h.glowText(m, m+80*s, 3.4*s, fmt.Sprintf("X%d COMBO", g.Combo), cAmber, 0)
		h.rect(m, m+108*s, 150*s*g.ComboT/3, 3*s, cAmber)
	}

	// wave
	wx := W - m // right edge of the wave block; with touch, PAUSE and VIEW sit in the corner
	if h.touch != nil {
		wx -= 152 * s
	}
	h.text(wx, m, 2*s, "WAVE", fade(cCyan, 0.8), 2)
	h.glowText(wx, m+18*s, 4.2*s, fmt.Sprintf("%02d", g.Wave), cWhite, 2)
	h.text(wx, m+56*s, 1.8*s, fmt.Sprintf("HOSTILES %d", len(g.Enemies)+len(g.queue)), fade(cCyan, 0.6), 2)
	h.text(wx, m+74*s, 1.8*s, diffs[g.Diff].name, fade(cAmber, 0.8), 2)

	// boss bar
	for _, e := range g.Enemies {
		if e.Kind == kBoss {
			bw := 420 * s
			h.text(W/2, m, 2*s, "DREADNOUGHT", cGreen, 1)
			h.rect(W/2-bw/2-2*s, m+20*s, bw+4*s, 12*s, fade(cGreen, 0.18))
			h.rect(W/2-bw/2, m+22*s, bw*e.HP/e.MaxHP, 8*s, cGreen)
		}
	}

	if g.State == stateRunning {
		// enemy markers
		for _, e := range g.Enemies {
			x, y, ok := r.project(e.Pos)
			d := e.Pos.Sub(r.camPos).Len()
			col := RGBA{kinds[e.Kind].col[0], kinds[e.Kind].col[1], kinds[e.Kind].col[2], 0.85}
			if !ok || x < 0 || x > W || y < 0 || y > H {
				// edge arrow pointing at off-screen threats
				dx, dy := x-W/2, y-H/2
				if !ok {
					rel := e.Pos.Sub(r.camPos)
					dx, dy = rel.Dot(r.camR), -rel.Dot(r.camU)
				}
				l := float32(math.Hypot(float64(dx), float64(dy)))
				if l < 1e-3 {
					continue
				}
				dx, dy = dx/l, dy/l
				rad := H * 0.40
				cx, cy := W/2+dx*rad, H/2+dy*rad
				a := 9 * s
				h.tri(cx+dx*a, cy+dy*a, cx-dx*a-dy*a*0.7, cy-dy*a+dx*a*0.7, cx-dx*a+dy*a*0.7, cy-dy*a-dx*a*0.7, col)
				continue
			}
			if d > 160 {
				b := (7 + kinds[e.Kind].radius*0.3) * s
				t := 1.5 * s
				for _, sx := range []float32{-1, 1} {
					for _, sy := range []float32{-1, 1} {
						h.line(x+sx*b, y+sy*b, x+sx*b*0.45, y+sy*b, t, col)
						h.line(x+sx*b, y+sy*b, x+sx*b, y+sy*b*0.45, t, col)
					}
				}
			}
		}

		// rocket lock: a diamond on the enemy a rocket would chase
		if g.Lock != nil && p.Rockets > 0 {
			if x, y, ok := r.project(g.Lock.Pos); ok {
				k := (16 + kinds[g.Lock.Kind].radius*0.4) * s
				lc := RGBA{1.0, 0.45, 0.35, 1}
				h.line(x, y-k, x+k, y, 2*s, lc)
				h.line(x+k, y, x, y+k, 2*s, lc)
				h.line(x, y+k, x-k, y, 2*s, lc)
				h.line(x-k, y, x, y-k, 2*s, lc)
				h.text(x, y+k+6*s, 1.6*s, "LOCK", lc, 1)
			}
		}
		// incoming rocket warning
		if n := g.HostileRockets(); n > 0 && blinkOn {
			h.glowText(W/2, H*0.34, 3.2*s, "ROCKET INCOMING", cRed, 1)
		}
		for i := range g.Rockets {
			rk := &g.Rockets[i]
			if !rk.Hostile {
				continue
			}
			if x, y, ok := r.project(rk.Pos); ok && x > 0 && x < W && y > 0 && y < H {
				h.arc(x, y, 12*s, 2*s, 0, 2*math.Pi, cRed)
			}
		}

		// crosshair at the convergence point
		if x, y, ok := r.project(p.Pos.Add(p.Front().Mul(600))); ok {
			c := cCyan
			if p.Overheated {
				c = cRed
			}
			if g.HitMark > 0 {
				c = cWhite
				k := 13 * s
				for _, sx := range []float32{-1, 1} {
					for _, sy := range []float32{-1, 1} {
						h.line(x+sx*k*0.5, y+sy*k*0.5, x+sx*k, y+sy*k, 2*s, cWhite)
					}
				}
			}
			h.arc(x, y, 15*s, 1.6*s, 0, 2*math.Pi, fade(c, 0.55))
			h.rect(x-1.5*s, y-1.5*s, 3*s, 3*s, c)
			for _, d := range [][2]float32{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				h.line(x+d[0]*20*s, y+d[1]*20*s, x+d[0]*28*s, y+d[1]*28*s, 2*s, c)
			}
			// heat wraps around the crosshair
			if p.Heat > 0.02 {
				hc := cAmber
				if p.Heat > 0.75 || p.Overheated {
					hc = cRed
				}
				h.arc(x, y, 36*s, 3*s, math.Pi/2+0.5, math.Pi/2+0.5+float64(p.Heat)*(2*math.Pi-1.0), hc)
			}
		}
	}

	// bottom centre: heat + boost (with touch, only in play: the menus need the room)
	if h.touch == nil || g.State == stateRunning {
		bw := 320 * s
		cx := W / 2
		bars := true
		if h.touch != nil {
			// between the joystick and the buttons; on a narrow screen there is
			// no room, and FIRE, BOOST and FLIP show the same on their rings
			hx, _ := stickHome(W, H, s)
			tx, _, tr := touchButton(tbTrack, W, H, s)
			lo, hi := hx+stickRadius*s+12*s, tx-tr-12*s
			bw, cx = min(bw, hi-lo), (lo+hi)/2
			bars = bw >= 170*s
		}
		bx, by := cx-bw/2, H-m-56*s
		heatCol := cCyan
		if p.Heat > 0.5 {
			heatCol = cAmber
		}
		if p.Heat > 0.78 || p.Overheated {
			heatCol = cRed
		}
		label := "WEAPON HEAT"
		if p.Overheated {
			label = "OVERHEATED - COOLING"
		}
		if bars {
			if !p.Overheated || blinkOn {
				h.text(cx, by-16*s, 1.8*s, label, heatCol, 1)
			}
			h.bar(bx, by, bw, 10*s, p.Heat, heatCol, 24)
			boost := float32(1)
			if p.BoostCD > 0 {
				boost = 1 - p.BoostCD/boostCD
			}
			bc := cAmber
			if boost < 1 {
				bc = fade(cAmber, 0.6)
			}
			h.rect(bx, by+16*s, bw, 4*s, fade(cAmber, 0.15))
			h.rect(bx, by+16*s, bw*boost, 4*s, bc)
			h.text(cx, by+24*s, 1.5*s, "BOOST", fade(cAmber, 0.8), 1)
			flip := float32(1)
			if p.FlipCD > 0 {
				flip = 1 - p.FlipCD/flipCD
			}
			fc := cGreen
			if flip < 1 {
				fc = fade(cGreen, 0.6)
			}
			h.rect(bx, by+40*s, bw, 4*s, fade(cGreen, 0.15))
			h.rect(bx, by+40*s, bw*flip, 4*s, fc)
			h.text(cx, by+48*s, 1.5*s, "FLIP", fade(cGreen, 0.8), 1)
		} else {
			cx, by = W/2, H*0.62 // the power-ups go above the controls
		}

		// active power-ups
		py := by - 40*s
		if p.RapidT > 0 {
			h.text(cx, py, 2*s, fmt.Sprintf("RAPID FIRE %.1f", p.RapidT), cAmber, 1)
			py -= 18 * s
		}
		if p.CoolT > 0 {
			h.text(cx, py, 2*s, fmt.Sprintf("COOLANT %.1f", p.CoolT), cCyan, 1)
		}

	}

	// bottom left (top left with touch, clear of the joystick): rockets and shields
	ly := H - m - 92*s
	if h.touch != nil {
		ly = m + 124*s
	}
	rc := RGBA{1.0, 0.45, 0.35, 1}
	h.text(m, ly, 1.8*s, "ROCKETS", fade(cCyan, 0.8), 0)
	for i := 0; i < maxRockets; i++ {
		c := fade(rc, 0.18)
		if i < p.Rockets {
			c = rc
		}
		x, y := m+float32(i)*13*s, ly+18*s
		h.rect(x, y+5*s, 8*s, 13*s, c)
		h.tri(x, y+5*s, x+8*s, y+5*s, x+4*s, y, c)
	}

	h.text(m, ly+48*s, 1.8*s, "SHIELDS", fade(cCyan, 0.8), 0)
	for i := 0; i < maxShields; i++ {
		x, y := m+float32(i)*30*s, ly+66*s
		c := fade(cCyan, 0.15)
		if i < p.Shields {
			c = cCyan
			if p.Shields <= 1 {
				c = cRed
				if !blinkOn {
					c = fade(cRed, 0.5)
				}
			} else if p.Shields == 2 {
				c = cAmber
			}
		}
		w, hh := 22*s, 26*s
		h.tri(x, y, x+w, y, x+w/2, y+hh, c)
		h.rect(x, y-5*s, w, 4*s, c)
	}

	// bottom right (top right with touch, under PAUSE and VIEW): radar
	rad := 78 * s
	cx, cy := W-m-rad, H-m-rad
	if h.touch != nil {
		rad = 52 * s
		cx, cy = W-m-rad, m+70*s+rad
	}
	h.disc(cx, cy, rad, RGBA{0.02, 0.08, 0.12, 0.55})
	h.arc(cx, cy, rad, 1.5*s, 0, 2*math.Pi, fade(cCyan, 0.7))
	h.arc(cx, cy, rad*0.5, 1*s, 0, 2*math.Pi, fade(cCyan, 0.25))
	h.line(cx-rad, cy, cx+rad, cy, 1*s, fade(cCyan, 0.18))
	h.line(cx, cy-rad, cx, cy+rad, 1*s, fade(cCyan, 0.18))
	sw := float64(g.Time) * 1.6
	h.line(cx, cy, cx+rad*float32(math.Cos(sw)), cy+rad*float32(math.Sin(sw)), 1.5*s, fade(cCyan, 0.4))
	f, rt := p.Front(), p.Right()
	flat := norm(V3{f[0], 0, f[2]})
	const rng = 900
	blip := func(pos V3, c RGBA, size float32) {
		rel := pos.Sub(p.Pos)
		x, z := rel.Dot(rt)/rng, rel.Dot(flat)/rng
		if l := float32(math.Hypot(float64(x), float64(z))); l > 0.94 {
			x, z = x/l*0.94, z/l*0.94
		}
		h.rect(cx+x*rad-size/2, cy-z*rad-size/2, size, size, c)
	}
	for i := range g.Asteroids {
		blip(g.Asteroids[i].Pos, RGBA{0.6, 0.6, 0.6, 0.35}, 2.5*s)
	}
	for _, pu := range g.Pickups {
		blip(pu.Pos, cGreen, 3.5*s)
	}
	for i := range g.Rockets {
		if g.Rockets[i].Hostile && blinkOn {
			blip(g.Rockets[i].Pos, cRed, 5*s)
		}
	}
	for _, e := range g.Enemies {
		k := kinds[e.Kind]
		size := 4 * s
		if e.Kind == kBoss {
			size = 9 * s
		}
		blip(e.Pos, RGBA{k.col[0], k.col[1], k.col[2], 1}, size)
	}
	h.tri(cx, cy-6*s, cx+4.5*s, cy+5*s, cx-4.5*s, cy+5*s, cWhite)

	// centre messages
	if g.BannerT > 0 && g.State == stateRunning {
		a := float32(math.Min(1, float64(g.BannerT)*2))
		h.glowText(W/2, H*0.24, 6*s, g.Banner, fade(cWhite, a), 1)
		if g.Sub != "" {
			h.text(W/2, H*0.24+56*s, 2.6*s, g.Sub, fade(cAmber, a), 1)
		}
	}
	if g.NoteT > 0 && g.State == stateRunning {
		a := float32(math.Min(1, float64(g.NoteT)*2))
		nc := cGreen
		if g.NoteBad {
			nc = cRed
		}
		h.text(W/2, H*0.66, 2.4*s, g.Note, fade(nc, a), 1)
	}
	if h.touch != nil && g.State == stateRunning {
		h.drawTouch(g, W, H)
	}
}

// drawTouch draws the phone controls: the joystick and the buttons, and the
// target TRACK is swinging onto.
func (h *HUD) drawTouch(g *Game, W, H float32) {
	t, s, p := h.touch, h.s, &g.P
	R := stickRadius * s
	a := float32(0.45)
	if t.StickOn {
		a = 0.9
	}
	h.disc(t.BaseX, t.BaseY, R, RGBA{0.02, 0.08, 0.12, 0.35 * a})
	h.arc(t.BaseX, t.BaseY, R, 2*s, 0, 2*math.Pi, fade(cCyan, 0.7*a))
	// ticks at the four directions, so the thumb's position reads at a glance
	for k := 0; k < 4; k++ {
		ang := float64(k) * math.Pi / 2
		cx, cy := float32(math.Cos(ang)), float32(math.Sin(ang))
		h.line(t.BaseX+cx*(R-12*s), t.BaseY+cy*(R-12*s), t.BaseX+cx*(R-4*s), t.BaseY+cy*(R-4*s), 2*s, fade(cCyan, 0.6*a))
	}
	// the boost zone, past the top of the ring: shown as the thumb heads there
	if t.StickOn && t.StickY > 0.5 {
		show := min((t.StickPush-0.8)/(stickBoost-0.8), 1)
		if show > 0 {
			bc := fade(cAmber, 0.35+0.4*show)
			if t.Boosting {
				bc = cAmber
			}
			// just above the knob at the top of its travel
			by := t.BaseY - R - 44*s
			for k := float32(0); k < 2; k++ {
				y := by - k*11*s
				h.line(t.BaseX-14*s, y+7*s, t.BaseX, y, 3*s, bc)
				h.line(t.BaseX, y, t.BaseX+14*s, y+7*s, 3*s, bc)
			}
			h.text(t.BaseX+22*s, by-9*s, 1.6*s, "BOOST", bc, 0)
		}
	}
	h.disc(t.KnobX, t.KnobY, 34*s, fade(cCyan, 0.20+0.30*a))
	h.arc(t.KnobX, t.KnobY, 34*s, 2*s, 0, 2*math.Pi, fade(cWhite, a))
	if !t.StickOn {
		h.text(t.BaseX, t.BaseY+R+8*s, 1.4*s, "MOVE", fade(cCyan, 0.5), 1)
	}

	// the enemy TRACK is turning toward
	if e := g.Tracked; e != nil {
		if x, y, ok := h.r.project(e.Pos); ok && x > 0 && x < W && y > 0 && y < H {
			k := (22 + kinds[e.Kind].radius*0.5) * s
			spin := float64(g.Time * 3)
			for q := 0; q < 4; q++ {
				a0 := spin + float64(q)*math.Pi/2
				h.arc(x, y, k, 2.5*s, a0, a0+0.8, cGreen)
			}
		}
	}

	pulse := float32(0.5 + 0.5*math.Sin(float64(g.Time*8)))
	for i := 0; i < tbCount; i++ {
		x, y, r := touchButton(i, W, H, s)
		c, label, px := cCyan, "", 1.8*s
		switch i {
		case tbFire:
			label, px = "FIRE", 2.8*s
			if p.Heat > 0.5 {
				c = cAmber
			}
			if p.Overheated || p.Heat > 0.78 {
				c = cRed
			}
		case tbRocket:
			label, px, c = fmt.Sprintf("ROCKET %d", p.Rockets), 1.5*s, RGBA{1.0, 0.45, 0.35, 1}
			if p.Rockets == 0 {
				c = fade(c, 0.45)
			}
		case tbBoost:
			label, c = "BOOST", cAmber
		case tbTrack:
			label, px, c = "TRACK", 1.5*s, cGreen
		case tbFlipL, tbFlipR:
			label, px, c = "FLIP", 1.7*s, cGreen
		case tbView:
			label, px = "VIEW", 1.5*s
		}
		fill := float32(0.16)
		if t.Held[i] {
			fill = 0.45
		}
		h.disc(x, y, r, fade(c, fill))
		h.arc(x, y, r, 2*s, 0, 2*math.Pi, fade(c, 0.85))
		switch i {
		case tbFire:
			// heat fills the ring, so it can be watched without looking away
			if p.Heat > 0.01 {
				h.arc(x, y, r-6*s, 4*s, -math.Pi/2, -math.Pi/2+2*math.Pi*float64(min(p.Heat, 1)), fade(c, 0.9))
			}
			// small arrows: the thumb can slide on FIRE to aim
			if !t.Held[i] {
				for k := 0; k < 4; k++ {
					ang := float64(k) * math.Pi / 2
					cx, cy := float32(math.Cos(ang)), float32(math.Sin(ang))
					ox, oy := x+cx*(r+9*s), y+cy*(r+9*s)
					h.tri(ox+cx*6*s, oy+cy*6*s, ox-cy*5*s, oy+cx*5*s, ox+cy*5*s, oy-cx*5*s, fade(c, 0.45))
				}
			}
			if g.AutoFire {
				h.text(x, y+r*0.42, 1.3*s, "AUTO", fade(cWhite, 0.75), 1)
			}
		case tbRocket:
			if g.Lock != nil && p.Rockets > 0 { // a rocket fired now would home
				h.arc(x, y, r+5*s+3*s*pulse, 2.5*s, 0, 2*math.Pi, fade(c, 0.6+0.4*pulse))
				h.text(x, y+r*0.38, 1.3*s, "LOCKED", c, 1)
			} else if p.rocketCD > 0 {
				h.arc(x, y, r-5*s, 3*s, -math.Pi/2, -math.Pi/2+2*math.Pi*float64(1-p.rocketCD/rocketCooldown), c)
			}
		case tbBoost: // cooldown ring
			if p.BoostCD > 0 {
				h.arc(x, y, r-5*s, 3*s, -math.Pi/2, -math.Pi/2+2*math.Pi*float64(1-p.BoostCD/boostCD), c)
			}
		case tbTrack: // a little crosshair over the label
			if g.Tracked != nil {
				h.arc(x, y, r+4*s, 2.5*s, 0, 2*math.Pi, fade(c, 0.6+0.4*pulse))
			}
			h.line(x-9*s, y-12*s, x+9*s, y-12*s, 2*s, fade(c, 0.9))
			h.line(x, y-21*s, x, y-3*s, 2*s, fade(c, 0.9))
			y += 8 * s
		case tbFlipL, tbFlipR:
			if p.FlipCD > 0 {
				h.arc(x, y, r-5*s, 3*s, -math.Pi/2, -math.Pi/2+2*math.Pi*float64(1-p.FlipCD/flipCD), c)
			}
			dir := float32(-1)
			if i == tbFlipR {
				dir = 1
			}
			ax := x + dir*r*0.55
			h.tri(ax+dir*7*s, y, ax-dir*3*s, y-7*s, ax-dir*3*s, y+7*s, fade(c, 0.9))
			x -= dir * 4 * s
		case tbPause:
			h.rect(x-9*s, y-11*s, 6*s, 22*s, fade(c, 0.9))
			h.rect(x+3*s, y-11*s, 6*s, 22*s, fade(c, 0.9))
		}
		if label != "" {
			h.text(x, y-3.5*px, px, label, fade(cWhite, 0.9), 1)
		}
	}
}
