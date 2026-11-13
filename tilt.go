package main

import "math"

// tiltSource reports which way is up relative to the screen: x to the
// screen's right, y to its top, z out of its face (any length).
type tiltSource interface {
	Up() (x, y, z float32, ok bool)
}

// Tilt turns how the phone is held into turning speed. The angle it is held
// at when a run starts (or when RECENTER is chosen) counts as level; tilting
// away from that keeps the ship turning, faster the further it is tilted.
//
//   - turning the phone like a steering wheel (right side down) turns right
//   - tilting its top edge away from you points the nose down
type Tilt struct {
	src        tiltSource
	ux, uy, uz float32 // smoothed up vector
	have       bool
	roll0      float32 // level, in degrees
	pitch0     float32
	recenter   bool
}

const (
	tiltDead = 4   // degrees off level that do nothing, so a steady hand holds course
	tiltFull = 30  // degrees off level for full turning speed
	tiltRate = 6.5 // turning speed per degree past the dead zone, degrees/second
)

func newTilt(src tiltSource) *Tilt {
	if src == nil {
		return nil
	}
	return &Tilt{src: src, recenter: true}
}

// Recenter makes the next reading the new level.
func (t *Tilt) Recenter() {
	if t != nil {
		t.recenter = true
	}
}

// angles returns the phone's roll (steering-wheel turn, right side down
// positive) and pitch (top edge away from you positive), in degrees.
func (t *Tilt) angles() (roll, pitch float32, ok bool) {
	x, y, z, ok := t.src.Up()
	if !ok || x*x+y*y+z*z < 1e-6 {
		return 0, 0, false
	}
	if !t.have {
		t.ux, t.uy, t.uz, t.have = x, y, z, true
	} else { // the accelerometer jitters; smooth it a little
		const k = 0.3
		t.ux += (x - t.ux) * k
		t.uy += (y - t.uy) * k
		t.uz += (z - t.uz) * k
	}
	roll = deg(math.Atan2(float64(-t.ux), math.Hypot(float64(t.uy), float64(t.uz))))
	pitch = deg(math.Atan2(float64(t.uz), float64(t.uy)))
	return roll, pitch, true
}

// Turn returns this frame's turn, in degrees, for Game.Look.
// speed scales it (1 is the default).
func (t *Tilt) Turn(dt, speed float32) (dx, dy float32) {
	roll, pitch, ok := t.angles()
	if !ok {
		return 0, 0
	}
	if t.recenter {
		t.roll0, t.pitch0, t.recenter = roll, pitch, false
	}
	return tiltSpeed(roll-t.roll0) * dt * speed, tiltSpeed(wrapDeg(pitch-t.pitch0)) * dt * speed
}

// tiltSpeed maps degrees off level to degrees per second of turning.
func tiltSpeed(a float32) float32 {
	m := float32(math.Abs(float64(a))) - tiltDead
	if m <= 0 {
		return 0
	}
	m = min(m, tiltFull-tiltDead)
	if a < 0 {
		m = -m
	}
	return m * tiltRate
}

func deg(r float64) float32 { return float32(r * 180 / math.Pi) }

// wrapDeg brings an angle difference into -180..180.
func wrapDeg(a float32) float32 {
	for a > 180 {
		a -= 360
	}
	for a < -180 {
		a += 360
	}
	return a
}

// Motion settings: what moving the phone does.
const (
	motionOff  = iota
	motionTilt // tilt away from level to keep turning (like a steering wheel)
	motionGyro // turn the phone and the ship turns with it (aiming like a camera)
)

var motionNames = [...]string{"OFF", "TILT", "GYRO"}

// gyroSource reports how fast the phone is rotating, in radians/second about
// the screen's axes: x to the right, y to the top, z out of its face.
type gyroSource interface {
	Rate() (x, y, z float32, ok bool)
}

// Gyro aims the way PUBG Mobile and Fortnite's gyro option do: turning the
// phone turns the ship by the same angle, times a gain. Turning left and
// right is measured about the real vertical (from the accelerometer), so it
// works the same whether the phone is held upright or tipped back.
type Gyro struct {
	src        gyroSource
	up         tiltSource // nil: use the screen's own vertical
	ux, uy, uz float32
	have       bool
}

const (
	gyroGain  = 2.2  // degrees of ship turn per degree of phone turn, at turn speed 5
	gyroNoise = 0.03 // rad/s below which rotation is sensor drift, not the player
)

func newGyro(src gyroSource, up tiltSource) *Gyro {
	if src == nil {
		return nil
	}
	return &Gyro{src: src, up: up}
}

// Turn returns this frame's turn, in degrees, for Game.Look.
func (g *Gyro) Turn(dt, speed float32) (dx, dy float32) {
	wx, wy, wz, ok := g.src.Rate()
	if !ok {
		return 0, 0
	}
	// which way is up, smoothed; held upright that is the screen's y
	ax, ay, az := float32(0), float32(1), float32(0)
	if g.up != nil {
		if x, y, z, ok := g.up.Up(); ok && x*x+y*y+z*z > 1e-6 {
			if !g.have {
				g.ux, g.uy, g.uz, g.have = x, y, z, true
			}
			const k = 0.1
			g.ux += (x - g.ux) * k
			g.uy += (y - g.uy) * k
			g.uz += (z - g.uz) * k
		}
		if g.have {
			l := float32(math.Sqrt(float64(g.ux*g.ux + g.uy*g.uy + g.uz*g.uz)))
			ax, ay, az = g.ux/l, g.uy/l, g.uz/l
		}
	}
	// counter-clockwise about up (seen from above) is turning left
	yaw := wx*ax + wy*ay + wz*az
	// about the screen's x: the top edge coming toward you looks up
	pitch := wx
	if math.Abs(float64(yaw)) < gyroNoise {
		yaw = 0
	}
	if math.Abs(float64(pitch)) < gyroNoise {
		pitch = 0
	}
	k := deg(1) * dt * gyroGain * speed
	return -yaw * k, -pitch * k
}
