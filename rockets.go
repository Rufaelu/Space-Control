package main

import (
	"math"
	"os"
	"path/filepath"
	"strings"
)

// --- difficulty levels ---

const (
	diffEasy = iota
	diffNormal
	diffHard
	diffCount
)

type DiffDef struct {
	name           string
	speed, fireGap float32 // enemy speed and time-between-shots multipliers
	rockets        int     // rockets the player starts with
	scoreX2        int     // score multiplier, in halves
	enemyRockets   bool
}

var diffs = [diffCount]DiffDef{
	{"EASY", 0.85, 1.5, 8, 1, false},
	{"NORMAL", 1.0, 1.0, 6, 2, false},
	{"HARD", 1.15, 0.7, 5, 4, true},
}

// --- pilot name ---

const maxNameLen = 12

func namePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "space-control", "name")
}

// filterName keeps only what the HUD font can draw, up to the length limit.
func filterName(s string) string {
	var b strings.Builder
	for _, ch := range strings.ToUpper(s) {
		if b.Len() >= maxNameLen {
			break
		}
		if (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == ' ' || ch == '-' {
			b.WriteRune(ch)
		}
	}
	return b.String()
}

// cleanName is filterName without surrounding spaces.
func cleanName(s string) string { return strings.TrimSpace(filterName(s)) }

func loadName() string {
	p := namePath()
	if p == "" {
		return ""
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return cleanName(strings.SplitN(string(b), "\n", 2)[0])
}

func saveName(n string) {
	p := namePath()
	if p == "" {
		return
	}
	if os.MkdirAll(filepath.Dir(p), 0o755) == nil {
		_ = os.WriteFile(p, []byte(n+"\n"), 0o644)
	}
}

var cheers = [...]string{"WELL DONE", "GREAT SHOOTING", "OUTSTANDING", "NICE FLYING", "SHARP WORK"}

// --- rockets ---

const (
	maxRockets     = 9
	rocketCooldown = 0.6
	rocketDamage   = 6 // direct hit
	rocketSplash   = 3 // everything else in the blast
	blastRadius    = 55
)

type Rocket struct {
	Pos, Prev, Dir V3
	Speed, Life    float32
	Hostile        bool
	Target         *Enemy
	trailAcc       float32
}

// pickLock returns the enemy closest to the crosshair, within range and a 20 degree cone.
func (g *Game) pickLock() *Enemy {
	p := &g.P
	f := p.Front()
	var best *Enemy
	bestDot := float32(0.94)
	for _, e := range g.Enemies {
		to := e.Pos.Sub(p.Pos)
		d := to.Len()
		if d > 1000 || d < 1 || e.HP <= 0 {
			continue
		}
		if dot := to.Mul(1 / d).Dot(f); dot > bestDot {
			best, bestDot = e, dot
		}
	}
	return best
}

func (g *Game) fireRocket() {
	p := &g.P
	f := p.Front()
	pos := p.Pos.Add(f.Mul(4)).Sub(p.Up().Mul(1.6))
	g.Rockets = append(g.Rockets, Rocket{Pos: pos, Prev: pos, Dir: f, Speed: p.Vel.Len() + 120, Life: 5, Target: g.Lock})
	p.Rockets--
	p.rocketCD = rocketCooldown
	sfx(sfxRocket, 1)
	g.buzz(0.45, 35)
	g.addPart(Particle{Pos: pos, Col: V3{1, 0.7, 0.3}, Size: 9, Life: 0.12, Max: 0.12, Bright: 4})
}

func (g *Game) enemyRocket(e *Enemy) {
	dir := norm(g.P.Pos.Sub(e.Pos))
	pos := e.Pos.Add(dir.Mul(kinds[e.Kind].radius))
	g.Rockets = append(g.Rockets, Rocket{Pos: pos, Prev: pos, Dir: dir, Speed: 90, Life: 7, Hostile: true})
	sfxPos(sfxRocket, 0.8, pos)
}

// HostileRockets counts incoming rockets, for the HUD warning.
func (g *Game) HostileRockets() int {
	n := 0
	for i := range g.Rockets {
		if g.Rockets[i].Hostile {
			n++
		}
	}
	return n
}

func (g *Game) updateRockets(dt float32) {
	p := &g.P
	running := g.State == stateRunning
	keep := g.Rockets[:0]
	incoming := false
	for _, r := range g.Rockets {
		r.Life -= dt
		r.Prev = r.Pos

		// steering
		var target V3
		homing := false
		maxSpeed, accel, turn := float32(460), float32(500), float32(5.0)
		if r.Hostile {
			maxSpeed, accel, turn = 175, 120, 1.5
			if running {
				target, homing = p.Pos, true
				incoming = true
			}
		} else if r.Target != nil && r.Target.HP > 0 {
			target, homing = r.Target.Pos, true
		}
		if homing {
			want := norm(target.Sub(r.Pos))
			r.Dir = norm(approach(r.Dir, want, turn*dt))
		}
		r.Speed = float32(math.Min(float64(maxSpeed), float64(r.Speed+accel*dt)))
		r.Pos = r.Pos.Add(r.Dir.Mul(r.Speed * dt))

		// exhaust
		col := V3{1, 0.65, 0.25}
		if r.Hostile {
			col = V3{1, 0.25, 0.2}
		}
		r.trailAcc += 90 * dt
		for ; r.trailAcc >= 1; r.trailAcc-- {
			g.addPart(Particle{Pos: r.Pos.Sub(r.Dir.Mul(3)), Vel: rndDir().Mul(3), Col: col,
				Size: 1.6, Grow: 3, Life: 0.55, Max: 0.55, Bright: 0.9})
		}

		// impacts
		boom := r.Life <= 0
		var direct *Enemy
		if !boom {
			for i := range g.Asteroids {
				a := &g.Asteroids[i]
				if segDist(r.Prev, r.Pos, a.Pos) < a.Scale*0.95 {
					boom = true
					if !r.Hostile {
						a.HP = 0 // a rocket shatters any rock outright
					}
					break
				}
			}
		}
		if !boom && r.Hostile {
			if running && segDist(r.Prev, r.Pos, p.Pos) < 7 {
				boom = true
				g.hurtPlayer()
			}
		} else if !boom {
			for _, e := range g.Enemies {
				if e.HP > 0 && segDist(r.Prev, r.Pos, e.Pos) < kinds[e.Kind].radius+2 {
					boom, direct = true, e
					break
				}
			}
		}
		if !boom {
			keep = append(keep, r)
			continue
		}
		if r.Hostile {
			g.explode(r.Pos, 1.0, V3{1, 0.3, 0.2})
			continue
		}
		g.explode(r.Pos, 2.4, V3{1, 0.6, 0.2})
		g.Shake = float32(math.Max(float64(g.Shake), 0.3))
		for _, e := range g.Enemies {
			if e.HP <= 0 {
				continue
			}
			if e == direct {
				e.HP -= rocketDamage
			} else if e.Pos.Sub(r.Pos).Len() < blastRadius+kinds[e.Kind].radius {
				e.HP -= rocketSplash
			} else {
				continue
			}
			e.Flash = 0.15
			g.HitMark = 0.14
		}
	}
	g.Rockets = keep

	// incoming-missile alarm
	g.alarmT -= dt
	if incoming && g.alarmT <= 0 {
		g.alarmT = 0.45
		sfx(sfxAlarm, 1)
	}
}
