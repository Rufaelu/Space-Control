package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/veandco/go-sdl2/sdl"
)

// Phone controls, held in landscape:
//
//   - a joystick on the left, which appears under the thumb: up flies forward,
//     down back, left and right slide. Pushing it hard forward, past its ring,
//     boosts (like sprinting in mobile shooters).
//   - dragging anywhere else on the screen turns the ship (like the mouse);
//     quick flicks turn further than slow, careful drags.
//   - FIRE also turns the ship as the thumb slides on it, so shooting never
//     stops the aiming.
//   - TRACK, held, swings the ship onto the nearest enemy, even behind it.
//   - a separate button for everything else.
//
// Layout is in HUD units (HUD.s), measured from the screen's corners.

const (
	tbFire = iota
	tbRocket
	tbBoost
	tbTrack
	tbFlipL
	tbFlipR
	tbView
	tbPause
	tbCount
)

// touchUI is how much larger the HUD is drawn with the phone controls: text
// sized for a monitor would be a millimetre tall on a phone.
const touchUI = 1.35

const (
	stickRadius = 88   // how far the knob travels
	stickDead   = 0.12 // fraction of the travel that does nothing
	stickBoost  = 1.3  // a push this far past the ring, straight ahead, boosts
	stickFollow = 1.7  // further than this, the stick follows the thumb
	stickZone   = 0.42 // share of the screen's width, from the left, that starts the stick
	dragTurn    = 0.22 // degrees of turn per HUD unit dragged, at turn speed 5
)

// touchButton is the centre and radius of one button.
func touchButton(i int, W, H, s float32) (x, y, r float32) {
	m := 28 * s
	switch i {
	case tbFire:
		return W - m - 92*s, H - m - 92*s, 72 * s
	case tbRocket:
		return W - m - 232*s, H - m - 46*s, 44 * s
	case tbBoost:
		return W - m - 228*s, H - m - 168*s, 44 * s
	case tbTrack:
		return W - m - 320*s, H - m - 100*s, 38 * s
	case tbFlipL:
		return W - m - 150*s, H - m - 252*s, 38 * s
	case tbFlipR:
		return W - m - 52*s, H - m - 252*s, 38 * s
	case tbView:
		return W - m - 104*s, m + 30*s, 30 * s
	default: // tbPause
		return W - m - 30*s, m + 30*s, 30 * s
	}
}

// stickHome is where the joystick sits when no thumb is on it.
func stickHome(W, H, s float32) (x, y float32) {
	return 28*s + 120*s, H - 28*s - 120*s
}

type finger struct {
	role   int // a tb* button, or roleStick / roleLook
	ox, oy float32
	x, y   float32
}

const (
	roleStick = tbCount + iota
	roleLook
	roleNone // landed while a menu was up; ignored until lifted
)

type Touch struct {
	fingers   map[sdl.FingerID]*finger
	lookX     float32 // drag since the last frame, HUD units
	lookY     float32
	tapped    [tbCount]bool // pressed since the last frame
	W, H, s   float32       // screen in HUD pixels, and the HUD scale
	StickOn   bool
	StickX    float32 // -1..1, right
	StickY    float32 // -1..1, forward
	StickPush float32 // how far the thumb is from the stick's centre, in ring radii
	Boosting  bool    // the stick is pushed hard forward
	BaseX     float32 // where the joystick is drawn
	BaseY     float32
	KnobX     float32
	KnobY     float32
	Held      [tbCount]bool
}

func newTouch() *Touch { return &Touch{fingers: map[sdl.FingerID]*finger{}} }

// layout records the screen size for this frame.
func (t *Touch) layout(W, H, s float32) {
	t.W, t.H, t.s = W, H, s
	if !t.StickOn {
		t.BaseX, t.BaseY = stickHome(W, H, s)
		t.KnobX, t.KnobY = t.BaseX, t.BaseY
	}
}

// buttonAt returns the button under a point, or -1. Buttons get a little
// extra margin, since a thumb is not precise; where margins overlap the
// nearest button wins.
func (t *Touch) buttonAt(x, y float32) int {
	best, bestD := -1, float32(math.MaxFloat32)
	for i := 0; i < tbCount; i++ {
		bx, by, r := touchButton(i, t.W, t.H, t.s)
		d := float32(math.Hypot(float64(x-bx), float64(y-by)))
		if d <= r*1.2 && d/r < bestD {
			best, bestD = i, d/r
		}
	}
	return best
}

// Down starts tracking a finger during play, and returns what it landed on:
// a tb* button, roleStick or roleLook.
func (t *Touch) Down(id sdl.FingerID, x, y float32) int {
	f := &finger{role: roleLook, ox: x, oy: y, x: x, y: y}
	if b := t.buttonAt(x, y); b >= 0 {
		f.role = b
		t.tapped[b] = true
	} else if x < t.W*stickZone && !t.StickOn {
		// the joystick appears under the thumb, kept fully on screen
		f.role = roleStick
		R := stickRadius * t.s
		t.StickOn = true
		t.BaseX = min(max(x, R+8*t.s), t.W*stickZone-R)
		t.BaseY = min(max(y, R+8*t.s), t.H-R-8*t.s)
		f.ox, f.oy = t.BaseX, t.BaseY
	}
	t.fingers[id] = f
	t.update()
	return f.role
}

// Ignore tracks a finger that landed on a menu, so lifting it later is harmless.
func (t *Touch) Ignore(id sdl.FingerID) { t.fingers[id] = &finger{role: roleNone} }

func (t *Touch) Move(id sdl.FingerID, x, y float32) {
	f := t.fingers[id]
	if f == nil {
		return
	}
	// a finger on FIRE aims too, so the thumb can shoot and steer at once
	if f.role == roleLook || f.role == tbFire {
		t.lookX += (x - f.x) / t.s
		t.lookY += (y - f.y) / t.s
	}
	f.x, f.y = x, y
	t.update()
}

func (t *Touch) Up(id sdl.FingerID) {
	if f := t.fingers[id]; f != nil && f.role == roleStick {
		t.StickOn = false
	}
	delete(t.fingers, id)
	t.update()
}

// Reset forgets every finger (when play stops).
func (t *Touch) Reset() {
	for id := range t.fingers {
		t.fingers[id] = &finger{role: roleNone}
	}
	t.StickOn, t.StickX, t.StickY, t.StickPush, t.Boosting = false, 0, 0, 0, false
	t.lookX, t.lookY = 0, 0
	t.tapped = [tbCount]bool{}
	t.update()
}

func (t *Touch) update() {
	t.Held = [tbCount]bool{}
	t.StickX, t.StickY, t.StickPush, t.Boosting = 0, 0, 0, false
	for _, f := range t.fingers {
		switch {
		case f.role < tbCount:
			t.Held[f.role] = true
		case f.role == roleStick:
			R := stickRadius * t.s
			dx, dy := f.x-f.ox, f.y-f.oy
			l := float32(math.Hypot(float64(dx), float64(dy)))
			// a thumb that wanders far off drags the stick along with it,
			// so turning back answers at once
			if l > stickFollow*R {
				k := l - stickFollow*R
				f.ox += dx / l * k
				f.oy += dy / l * k
				t.BaseX, t.BaseY = f.ox, f.oy
				dx, dy, l = f.x-f.ox, f.y-f.oy, stickFollow*R
			}
			t.StickPush = l / R
			// pushed hard, straight ahead: boost
			t.Boosting = l > stickBoost*R && -dy > l*0.82
			if l > R {
				dx, dy = dx/l*R, dy/l*R
				l = R
			}
			t.KnobX, t.KnobY = t.BaseX+dx, t.BaseY+dy
			// past the dead zone, the output starts from zero rather than
			// jumping, so small corrections stay small
			if n := l / R; n > stickDead {
				k := (n - stickDead) / (1 - stickDead) / n
				t.StickX, t.StickY = dx/R*k, -dy/R*k
			}
		}
	}
	if !t.StickOn {
		t.KnobX, t.KnobY = t.BaseX, t.BaseY
	}
}

// Look returns and clears the drag since the last frame, in degrees of turn.
// Slow drags turn a little less than they move, for fine aim; quick flicks
// turn further, so swinging round does not take several swipes.
func (t *Touch) Look(dt, speed float32) (dx, dy float32) {
	l := float32(math.Hypot(float64(t.lookX), float64(t.lookY)))
	accel := float32(1)
	if dt > 0 {
		v := l / dt // HUD units per second
		accel = 0.8 + 0.7*min(max((v-250)/1500, 0), 1)
	}
	k := dragTurn * speed * accel
	dx, dy = t.lookX*k, t.lookY*k
	t.lookX, t.lookY = 0, 0
	return dx, dy
}

// Tapped reports whether a button was pressed since the last call.
func (t *Touch) Tapped(b int) bool {
	v := t.tapped[b]
	t.tapped[b] = false
	return v
}

// --- saved settings: tilt on/off and turn speed ---

func controlsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "space-control", "controls")
}

func loadControls(g *Game) {
	b, err := os.ReadFile(controlsPath())
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(b), "\n") {
		k, v, _ := strings.Cut(strings.TrimSpace(line), "=")
		n, err := strconv.Atoi(v)
		if err != nil {
			continue
		}
		switch k {
		case "tilt": // from before the motion setting: on or off
			g.Motion = motionOff
			if n != 0 {
				g.Motion = motionTilt
			}
		case "motion":
			g.Motion = min(max(n, motionOff), motionGyro)
		case "turn":
			g.TurnSpeed = min(max(n, 1), 10)
		case "assist":
			g.Assist = n != 0
		case "autofire":
			g.AutoFire = n != 0
		case "vibrate":
			g.Vibrate = n != 0
		}
	}
}

func saveControls(g *Game) {
	p := controlsPath()
	if p == "" || os.MkdirAll(filepath.Dir(p), 0o755) != nil {
		return
	}
	b := func(v bool) int {
		if v {
			return 1
		}
		return 0
	}
	_ = os.WriteFile(p, []byte(fmt.Sprintf("motion=%d\nturn=%d\nassist=%d\nautofire=%d\nvibrate=%d\n",
		g.Motion, g.TurnSpeed, b(g.Assist), b(g.AutoFire), b(g.Vibrate))), 0o644)
}
