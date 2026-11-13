package main

import (
	"math"

	mgl "github.com/go-gl/mathgl/mgl32"
	"github.com/veandco/go-sdl2/sdl"
)

// Help for aiming with thumbs, which are far less precise than a mouse.
// Mobile shooters do the same: the turn slows down while the crosshair is
// over a target ("friction"), and the nose is nudged onto a target near the
// crosshair while the player is already turning, moving or firing
// ("magnetism"), never while they are hands-off.

const (
	assistCone   = 7.0  // degrees off the crosshair a target gets pulled from
	assistPull   = 4.5  // pull, as a fraction of the error per second
	assistMax    = 30.0 // fastest pull, degrees/second
	frictionIn   = 3.5  // degrees: full slowdown inside this
	frictionOut  = 9.0  // degrees: no slowdown outside this
	frictionLow  = 0.45 // turn speed kept over a target
	assistRange  = 900.0
	autoFireCone = 2.2 // degrees beyond an enemy's own size that still fire
	autoFireHeat = 0.8 // auto fire waits above this, so it never overheats
	trackRate    = 240.0
	trackRange   = 1500.0
)

// aimErr returns how far the ship must turn, in yaw and pitch degrees, to
// point straight at pos, and the distance to it.
func (g *Game) aimErr(pos V3) (yaw, pitch, dist float32) {
	p := &g.P
	to := pos.Sub(p.Pos)
	dist = to.Len()
	if dist < 1e-3 {
		return 0, 0, 0
	}
	to = to.Mul(1 / dist)
	wantYaw := deg(math.Atan2(float64(to.Z()), float64(to.X())))
	wantPitch := deg(math.Asin(float64(min(max(to.Y(), -1), 1))))
	return wrapDeg(wantYaw - p.Yaw), wantPitch - p.Pitch, dist
}

// offAxis is the angle between the crosshair and pos, in degrees.
func (g *Game) offAxis(pos V3) (angle, dist float32) {
	to := pos.Sub(g.P.Pos)
	dist = to.Len()
	if dist < 1e-3 {
		return 0, 0
	}
	c := to.Mul(1 / dist).Dot(g.P.Front())
	return deg(math.Acos(float64(min(max(c, -1), 1)))), dist
}

// nearCrosshair returns the living enemy closest to the crosshair within cone
// degrees and range, and its angle off the crosshair.
func (g *Game) nearCrosshair(cone, rng float32) (*Enemy, float32) {
	var best *Enemy
	bestA := cone
	for _, e := range g.Enemies {
		if e.HP <= 0 {
			continue
		}
		if a, d := g.offAxis(e.Pos); d < rng && a < bestA {
			best, bestA = e, a
		}
	}
	return best, bestA
}

// AssistFriction is how much of a drag's turn to keep this frame: less while
// an enemy sits under the crosshair, so the thumb does not overshoot it.
func (g *Game) AssistFriction() float32 {
	if _, a := g.nearCrosshair(frictionOut, assistRange); a < frictionOut {
		k := mgl.Clamp((a-frictionIn)/(frictionOut-frictionIn), 0, 1)
		return frictionLow + (1-frictionLow)*k
	}
	return 1
}

// AssistPull nudges the nose toward the enemy nearest the crosshair.
func (g *Game) AssistPull(dt float32) {
	e, _ := g.nearCrosshair(assistCone, assistRange)
	if e == nil {
		return
	}
	dy, dp, _ := g.aimErr(e.Pos)
	g.turnToward(dy, dp, assistPull*dt, assistMax*dt)
}

// turnToward turns by frac of the given error, at most maxStep degrees.
func (g *Game) turnToward(dYaw, dPitch, frac, maxStep float32) {
	sx, sy := dYaw*min(frac, 1), dPitch*min(frac, 1)
	if l := float32(math.Hypot(float64(sx), float64(sy))); l > maxStep {
		sx, sy = sx/l*maxStep, sy/l*maxStep
	}
	g.Look(sx, -sy)
}

// AutoFireWant reports whether the crosshair is on an enemy within gun range
// and the guns are cool enough to keep shooting.
func (g *Game) AutoFireWant() bool {
	p := &g.P
	if p.Overheated || p.Heat > autoFireHeat {
		return false
	}
	for _, e := range g.Enemies {
		if e.HP <= 0 {
			continue
		}
		a, d := g.offAxis(e.Pos)
		if d > 650 || d < 1 {
			continue
		}
		size := deg(math.Atan(float64(kinds[e.Kind].radius / d)))
		if a < size+autoFireCone {
			return true
		}
	}
	return false
}

// TrackTurn swings the ship onto an enemy, wherever it is: the one nearest
// the crosshair, with distance counting a little. It keeps the same target
// until it dies or the button is let go.
func (g *Game) TrackTurn(dt, speed float32) {
	alive := false
	for _, e := range g.Enemies {
		if e == g.Tracked && e.HP > 0 {
			alive = true
		}
	}
	if !alive {
		g.Tracked = nil
		best := float32(math.MaxFloat32)
		for _, e := range g.Enemies {
			if e.HP <= 0 {
				continue
			}
			a, d := g.offAxis(e.Pos)
			if d > trackRange {
				continue
			}
			if score := a + d*0.04; score < best {
				g.Tracked, best = e, score
			}
		}
	}
	if g.Tracked == nil {
		return
	}
	dy, dp, _ := g.aimErr(g.Tracked.Pos)
	g.turnToward(dy, dp, 7*dt, trackRate*max(speed, 0.6)*dt)
}

// --- vibration ---

var haptic *sdl.Haptic

// openHaptic finds the phone's vibrator (nothing on a desktop).
func openHaptic() {
	if sdl.InitSubSystem(sdl.INIT_HAPTIC) != nil {
		return
	}
	n, _ := sdl.NumHaptics()
	for i := 0; i < n; i++ {
		h, err := sdl.HapticOpen(i)
		if err != nil {
			continue
		}
		if h.RumbleInit() == nil {
			haptic = h
			return
		}
		h.Close()
	}
}

// buzz vibrates the phone, if it can and vibration is on. strength is 0..1.
func (g *Game) buzz(strength float32, ms uint32) {
	if haptic != nil && g.Vibrate && g.State != statePaused {
		haptic.RumblePlay(strength, ms)
	}
}
