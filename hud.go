package main

import (
	"fmt"
	"math"

	"github.com/go-gl/gl/v4.1-core/gl"
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
	r *Renderer
	s float32 // UI scale
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
	h.s = H / 720
	s := h.s
	blinkOn := int(g.Time*6)%2 == 0

	switch g.State {
	case stateMenu:
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
		qx, qy, qw, qh := titleQuitRect(W, H)
		if g.QuitHover {
			h.rect(qx, qy, qw, qh, fade(cRed, 0.30))
			h.rect(qx, qy+qh-3*s, qw, 3*s, cRed)
			h.glowText(qx+qw/2, qy+qh/2-7*s, 2.2*s, "QUIT GAME", cWhite, 1)
		} else {
			h.rect(qx, qy, qw, qh, RGBA{0, 0, 0, 0.45})
			h.rect(qx, qy+qh-2*s, qw, 2*s, fade(cCyan, 0.5))
			h.text(qx+qw/2, qy+qh/2-7*s, 2.2*s, "QUIT GAME", fade(cCyan, 0.8), 1)
		}
		if g.Track != "" {
			h.text(28*s, H-34*s, 1.8*s, "NOW PLAYING  "+g.Track, fade(cAmber, 0.8), 0)
		}
	case stateName:
		h.glowText(W/2, H*0.16, 11*s, "SPACE CONTROL", cCyan, 1)
		h.rect(0, H*0.40, W, H*0.30, RGBA{0, 0, 0, 0.5})
		h.text(W/2, H*0.44, 3*s, "WHAT IS YOUR NAME, PILOT?", cWhite, 1)
		typed := g.NameBuf
		if blinkOn && len(typed) < maxNameLen {
			typed += "_"
		}
		h.glowText(W/2-textW(g.NameBuf, 6*s)/2, H*0.52, 6*s, typed, cAmber, 0)
		h.text(W/2, H*0.63, 2*s, "TYPE YOUR NAME      ENTER - CONFIRM", fade(cCyan, 0.8), 1)
	default:
		h.drawPlay(g, W, H, blinkOn)
	}

	if g.State == statePaused {
		h.glowText(W/2, H*0.20, 8*s, "PAUSED", cWhite, 1)
		music := "MUSIC  OFF"
		if g.Music {
			music = "MUSIC  ON"
		}
		labels := [pauseCount]string{"RESUME", "RESTART", fmt.Sprintf("VOLUME  %d/10", g.Volume), fmt.Sprintf("ENGINES  %d/10", g.EngineVol), music, "QUIT TO TITLE", "QUIT GAME"}
		for i, l := range labels {
			x, y, w, hh := pauseItemRect(i, W, H)
			if i == g.MenuSel {
				h.rect(x, y, w, hh, fade(cCyan, 0.18))
				h.rect(x, y, 4*s, hh, cCyan)
				h.rect(x+w-4*s, y, 4*s, hh, cCyan)
				h.glowText(W/2, y+hh/2-10.5*s, 3*s, l, cWhite, 1)
			} else {
				h.text(W/2, y+hh/2-10.5*s, 3*s, l, fade(cCyan, 0.75), 1)
			}
			if i == pauseVolume || i == pauseEngine {
				// level pips under the label
				level := g.Volume
				if i == pauseEngine {
					level = g.EngineVol
				}
				pw := 14 * s
				for k := 0; k < 10; k++ {
					c := fade(cCyan, 0.2)
					if k < level {
						c = cAmber
					}
					h.rect(W/2-5*pw+float32(k)*pw+1*s, y+hh-5*s, pw-2*s, 3*s, c)
				}
			}
		}
		h.text(W/2, H*0.80, 1.8*s, "W / S OR MOUSE - CHOOSE      ENTER OR CLICK - SELECT      A / D - ADJUST      ESC - RESUME", fade(cCyan, 0.7), 1)
	}
	if g.State == stateGameOver {
		h.rect(0, H*0.30, W, H*0.36, RGBA{0, 0, 0, 0.45})
		h.glowText(W/2, H*0.34, 8*s, "SHIP DESTROYED", cRed, 1)
		h.text(W/2, H*0.34+78*s, 3.4*s, fmt.Sprintf("SCORE %06d", g.Score), cWhite, 1)
		h.text(W/2, H*0.34+112*s, 2.4*s, fmt.Sprintf("WAVE %d      KILLS %d", g.Wave, g.Kills), cCyan, 1)
		if g.NewHi {
			h.glowText(W/2, H*0.34+142*s, 2.8*s, "NEW BEST!", cAmber, 1)
		} else {
			h.text(W/2, H*0.34+142*s, 2.2*s, fmt.Sprintf("BEST %06d", g.HiScore), fade(cWhite, 0.7), 1)
		}
		if blinkOn {
			h.text(W/2, H*0.34+182*s, 2.6*s, "R - FLY AGAIN      ESC - TITLE", cAmber, 1)
		}
	}

	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	r.hud.use()
	r.hud.f2("uRes", W, H)
	r.hudB.flush()
	gl.Disable(gl.BLEND)
}

// titleQuitRect is the QUIT GAME button in the title screen's top-right corner.
func titleQuitRect(W, H float32) (x, y, w, h float32) {
	s := H / 720
	w, h = 160*s, 38*s
	return W - w - 24*s, 24 * s, w, h
}

func inRect(px, py, x, y, w, h float32) bool { return px >= x && px <= x+w && py >= y && py <= y+h }

// pauseItemRect is the on-screen box of pause-menu entry i.
func pauseItemRect(i int, W, H float32) (x, y, w, h float32) {
	s := H / 720
	w, h = 380*s, 40*s
	return W/2 - w/2, H*0.31 + float32(i)*46*s, w, h
}

// pauseItemAt returns the pause-menu entry under a pixel, or -1.
func pauseItemAt(px, py, W, H float32) int {
	for i := 0; i < pauseCount; i++ {
		if x, y, w, h := pauseItemRect(i, W, H); px >= x && px <= x+w && py >= y && py <= y+h {
			return i
		}
	}
	return -1
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
	h.text(W-m, m, 2*s, "WAVE", fade(cCyan, 0.8), 2)
	h.glowText(W-m, m+18*s, 4.2*s, fmt.Sprintf("%02d", g.Wave), cWhite, 2)
	h.text(W-m, m+56*s, 1.8*s, fmt.Sprintf("HOSTILES %d", len(g.Enemies)+len(g.queue)), fade(cCyan, 0.6), 2)
	h.text(W-m, m+74*s, 1.8*s, diffs[g.Diff].name, fade(cAmber, 0.8), 2)

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

	// bottom centre: heat + boost
	bw := 320 * s
	bx, by := W/2-bw/2, H-m-56*s
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
	if !p.Overheated || blinkOn {
		h.text(W/2, by-16*s, 1.8*s, label, heatCol, 1)
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
	h.text(W/2, by+24*s, 1.5*s, "BOOST", fade(cAmber, 0.8), 1)
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
	h.text(W/2, by+48*s, 1.5*s, "FLIP", fade(cGreen, 0.8), 1)

	// active power-ups
	py := by - 40*s
	if p.RapidT > 0 {
		h.text(W/2, py, 2*s, fmt.Sprintf("RAPID FIRE %.1f", p.RapidT), cAmber, 1)
		py -= 18 * s
	}
	if p.CoolT > 0 {
		h.text(W/2, py, 2*s, fmt.Sprintf("COOLANT %.1f", p.CoolT), cCyan, 1)
	}

	// bottom left: rockets
	rc := RGBA{1.0, 0.45, 0.35, 1}
	h.text(m, H-m-92*s, 1.8*s, "ROCKETS", fade(cCyan, 0.8), 0)
	for i := 0; i < maxRockets; i++ {
		c := fade(rc, 0.18)
		if i < p.Rockets {
			c = rc
		}
		x, y := m+float32(i)*13*s, H-m-74*s
		h.rect(x, y+5*s, 8*s, 13*s, c)
		h.tri(x, y+5*s, x+8*s, y+5*s, x+4*s, y, c)
	}

	// bottom left: shields
	h.text(m, H-m-44*s, 1.8*s, "SHIELDS", fade(cCyan, 0.8), 0)
	for i := 0; i < maxShields; i++ {
		x, y := m+float32(i)*30*s, H-m-26*s
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

	// bottom right: radar
	rad := 78 * s
	cx, cy := W-m-rad, H-m-rad
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
}
