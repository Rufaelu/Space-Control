package main

import (
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	mgl "github.com/go-gl/mathgl/mgl32"
)

type GameState int

const (
	stateMenu GameState = iota
	stateRunning
	statePaused
	stateGameOver
	stateName // typing the pilot name
)

const (
	maxShields   = 5
	playerSpeed  = 130.0
	boostSpeed   = 330.0
	boostTime    = 1.3
	boostCD      = 5.0
	flipTime     = 0.45
	flipCD       = 1.5
	flipSpeed    = 300.0 // sideways speed during a flip
	laserSpeed   = 720.0
	fireRate     = 0.12
	heatPerShot  = 0.085
	coolRate     = 0.40
	fieldHalf    = 900.0
	numAsteroids = 36
	maxParticles = 7000
)

// Pause menu entries.
const (
	pauseResume = iota
	pauseRestart
	pauseVolume
	pauseEngine
	pauseMusic
	pauseTitle
	pauseQuit
	pauseCount
)

const (
	kDrone = iota
	kFighter
	kBruiser
	kBoss
)

type KindDef struct {
	scale, radius, hp, speed float32
	score                    int
	col                      V3
	tail                     float32 // engine offset behind centre, model units
}

var kinds = [...]KindDef{
	kDrone:   {2.0, 10, 1, 58, 10, V3{1.0, 0.30, 0.12}, 3.8},
	kFighter: {2.0, 11, 2, 72, 25, V3{1.0, 0.20, 0.9}, 5.2},
	kBruiser: {2.2, 19, 7, 30, 60, V3{1.0, 0.48, 0.08}, 6.6},
	kBoss:    {3.0, 42, 70, 45, 500, V3{0.28, 1.0, 0.42}, 13.5},
}

const (
	puShield = iota
	puCoolant
	puRapid
	puRocket
)

var puCols = [...]V3{{0.3, 1.0, 0.45}, {0.3, 0.8, 1.0}, {1.0, 0.6, 0.15}, {1.0, 0.25, 0.4}}
var puNames = [...]string{"SHIELD RESTORED", "COOLANT FLUSH", "RAPID FIRE", "ROCKETS FULL"}

type Player struct {
	Pos, Vel         V3
	Yaw, Pitch, Roll float32
	yawRate, pitRate float32
	Shields          int
	Invuln           float32
	Heat             float32
	Overheated       bool
	BoostT, BoostCD  float32
	RapidT, CoolT    float32
	fireCD           float32
	trailAcc         float32
	FlipT, FlipCD    float32 // flip: a sideways barrel-roll dodge
	Rockets          int
	rocketCD         float32
	flipDir          float32 // -1 flips round to the left, +1 to the right
}

// flipK is the eased 0..1 progress of the current flip (0 when not flipping).
func (p *Player) flipK() float32 {
	if p.FlipT <= 0 {
		return 0
	}
	k := 1 - p.FlipT/flipTime
	return k * k * (3 - 2*k)
}

// bank is the ship's total roll angle, including the flip's barrel roll.
func (p *Player) bank() float32 { return p.Roll - p.flipDir*2*math.Pi*p.flipK() }

func (p *Player) Front() V3 {
	y, pt := float64(mgl.DegToRad(p.Yaw)), float64(mgl.DegToRad(p.Pitch))
	return V3{float32(math.Cos(y) * math.Cos(pt)), float32(math.Sin(pt)), float32(math.Sin(y) * math.Cos(pt))}
}
func (p *Player) Right() V3 { return norm(p.Front().Cross(V3{0, 1, 0})) }
func (p *Player) Up() V3    { return norm(p.Right().Cross(p.Front())) }

// orient builds a rotation taking model -Z to fwd.
func orient(fwd, upHint V3) mgl.Mat4 {
	r := fwd.Cross(upHint)
	if r.Len() < 1e-4 {
		r = fwd.Cross(V3{1, 0, 0})
	}
	r = r.Normalize()
	u := r.Cross(fwd).Normalize()
	return mgl.Mat4FromCols(r.Vec4(0), u.Vec4(0), fwd.Mul(-1).Vec4(0), mgl.Vec4{0, 0, 0, 1})
}

func (p *Player) Model() mgl.Mat4 {
	lead := mgl.HomogRotate3DY(mgl.DegToRad(-p.yawRate * 0.5)).Mul4(mgl.HomogRotate3DX(mgl.DegToRad(p.pitRate * 0.5)))
	return mgl.Translate3D(p.Pos[0], p.Pos[1], p.Pos[2]).
		Mul4(orient(p.Front(), V3{0, 1, 0})).Mul4(lead).Mul4(mgl.HomogRotate3DZ(p.bank()))
}

type Enemy struct {
	Kind            int
	Pos, Vel, Fwd   V3
	HP, MaxHP       float32
	Flash           float32
	fireT, phase    float32
	strafe          V3
	strafeT, spawnT float32
	rocketT         float32
	trailAcc        float32
	Age             float32
}

type Shot struct {
	Pos, Prev, Vel V3
	Life           float32
	Hostile        bool
	Col            V3
}

type Particle struct {
	Pos, Vel                V3
	Col                     V3
	Size, Grow              float32
	Life, Max, Drag, Bright float32
	Shape                   float32
}

type Pickup struct {
	Pos  V3
	Kind int
	Life float32
}

type Asteroid struct {
	Pos, Vel, Axis   V3
	Ang, Spin, Scale float32
	Mesh             int
	HP               int // bullet hits left before it breaks apart
}

// rockHP is how many bullet hits a rock of this size takes.
func rockHP(scale float32) int { return 2 + int(scale/9) }

// newRock makes a drifting rock of a random size at pos.
func newRock(pos V3, mesh int) Asteroid {
	a := Asteroid{Pos: pos, Vel: rndDir().Mul(2 + rnd()*6), Axis: rndDir(), Spin: 0.1 + rnd()*0.5,
		Scale: 9 + rnd()*rnd()*46, Mesh: mesh % 4}
	a.HP = rockHP(a.Scale)
	return a
}

// breakRocks shatters every rock that has run out of hit points: big ones split
// into smaller pieces, and a fresh rock drifts in far away to keep the field full.
func (g *Game) breakRocks() {
	keep := g.Asteroids[:0]
	var born []Asteroid
	for _, a := range g.Asteroids {
		if a.HP > 0 {
			keep = append(keep, a)
			continue
		}
		g.explode(a.Pos, a.Scale/16, V3{0.9, 0.7, 0.5})
		if g.State == stateRunning {
			g.Score += 5
		}
		if a.Scale > 18 {
			for i := 0; i < 2+rand.Intn(2); i++ {
				d := rndDir()
				f := Asteroid{Pos: a.Pos.Add(d.Mul(a.Scale * 0.5)), Vel: a.Vel.Add(d.Mul(14 + rnd()*14)),
					Axis: rndDir(), Spin: 0.4 + rnd()*1.2, Scale: a.Scale * (0.38 + rnd()*0.14), Mesh: rand.Intn(4)}
				f.HP = rockHP(f.Scale)
				born = append(born, f)
			}
		}
	}
	g.Asteroids = append(keep, born...)
	for len(g.Asteroids) < numAsteroids {
		g.Asteroids = append(g.Asteroids, newRock(g.P.Pos.Add(rndDir().Mul(fieldHalf*0.9)), rand.Intn(4)))
	}
}

type Input struct {
	Fwd, Strafe  float32
	Fire, Boost  bool
	FlipL, FlipR bool
	Rocket       bool
}

type Game struct {
	State     GameState
	P         Player
	Enemies   []*Enemy
	Shots     []Shot
	Parts     []Particle
	Pickups   []Pickup
	Asteroids []Asteroid
	Dust      []V3

	Score, HiScore int
	Combo          int
	ComboT         float32
	Wave           int
	queue          []int
	spawnT, waveT  float32
	Banner, Sub    string
	BannerT        float32
	Note           string
	NoteT          float32
	NoteBad        bool
	MenuSel        int  // highlighted pause-menu entry
	Cockpit        bool // first-person view
	Diff           int  // difficulty level
	Name           string
	NameBuf        string // text being typed on the name screen
	Rockets        []Rocket
	Lock           *Enemy // enemy a rocket would home on
	alarmT         float32
	Volume         int // 0..10
	Music          bool
	EngineVol      int // 0..10, loudness of ship engines
	Shake, HitFx   float32
	HitMark        float32
	Time           float32
	NewHi          bool
	Kills          int
}

func rnd() float32  { return rand.Float32() }
func rndS() float32 { return rand.Float32()*2 - 1 }
func rndDir() V3 {
	for {
		v := V3{rndS(), rndS(), rndS()}
		if l := v.Len(); l > 0.05 && l <= 1 {
			return v.Mul(1 / l)
		}
	}
}

func hiScorePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "space-control", "highscore")
}

func loadHiScore() int {
	p := hiScorePath()
	if p == "" {
		return 0
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	return n
}

func saveHiScore(n int) {
	p := hiScorePath()
	if p == "" {
		return
	}
	if os.MkdirAll(filepath.Dir(p), 0o755) == nil {
		_ = os.WriteFile(p, []byte(strconv.Itoa(n)), 0o644)
	}
}

func NewGame() *Game {
	g := &Game{HiScore: loadHiScore(), Volume: 7, EngineVol: 10, Music: true, Diff: diffNormal, Name: loadName()}
	for i := 0; i < 420; i++ {
		g.Dust = append(g.Dust, V3{rnd() * 400, rnd() * 400, rnd() * 400})
	}
	g.Reset()
	g.State = stateMenu
	if g.Name == "" {
		g.State = stateName
	}
	return g
}

func (g *Game) Reset() {
	g.P = Player{Yaw: -90, Shields: maxShields, Rockets: diffs[g.Diff].rockets}
	g.Enemies, g.Shots, g.Parts, g.Pickups = nil, nil, nil, nil
	g.Rockets, g.Lock = nil, nil
	g.Score, g.Combo, g.ComboT, g.Wave, g.Kills = 0, 0, 0, 0, 0
	g.queue, g.waveT, g.spawnT = nil, 2.0, 0
	g.Banner, g.BannerT, g.Note, g.NoteT = "", 0, "", 0
	g.Shake, g.HitFx, g.NewHi = 0, 0, false
	g.Asteroids = g.Asteroids[:0]
	for i := 0; i < numAsteroids; i++ {
		a := newRock(V3{rndS() * fieldHalf, rndS() * fieldHalf, rndS() * fieldHalf}, i)
		if a.Pos.Len() < 220 {
			a.Pos = norm(a.Pos).Mul(220 + rnd()*300)
		}
		g.Asteroids = append(g.Asteroids, a)
	}
	g.State = stateRunning
}

func (g *Game) banner(main, sub string, t float32) { g.Banner, g.Sub, g.BannerT = main, sub, t }

func (g *Game) startWave() {
	g.Wave++
	n := g.Wave
	g.queue = g.queue[:0]
	add := func(k, c int) {
		for i := 0; i < c; i++ {
			g.queue = append(g.queue, k)
		}
	}
	if n%5 == 0 {
		add(kDrone, min(4+n, 16))
		add(kFighter, min(n/3, 6))
		rand.Shuffle(len(g.queue), func(i, j int) { g.queue[i], g.queue[j] = g.queue[j], g.queue[i] })
		g.queue = append([]int{kBoss}, g.queue...)
		g.banner("WAVE "+strconv.Itoa(n), "DREADNOUGHT INBOUND", 3.5)
		sfx(sfxWave, 1)
		return
	}
	add(kDrone, min(4+2*n, 22))
	add(kFighter, min(n, 9))
	add(kBruiser, min(max((n-1)/2, 0), 5))
	rand.Shuffle(len(g.queue), func(i, j int) { g.queue[i], g.queue[j] = g.queue[j], g.queue[i] })
	g.banner("WAVE "+strconv.Itoa(n), strconv.Itoa(len(g.queue))+" HOSTILES", 3.0)
	sfx(sfxWave, 1)
}

func (g *Game) spawnEnemy(kind int, pos V3) {
	d := kinds[kind]
	hp := d.hp
	if kind == kBoss {
		hp += float32(g.Wave/5-1) * 35
	}
	g.Enemies = append(g.Enemies, &Enemy{
		Kind: kind, Pos: pos, Fwd: norm(g.P.Pos.Sub(pos)), HP: hp, MaxHP: hp,
		phase: rnd() * 6.28, fireT: 1.5 + rnd()*2, strafe: rndDir(), rocketT: 3 + rnd()*5,
	})
	// warp-in flash
	g.addPart(Particle{Pos: pos, Col: d.col, Size: d.radius * 3, Grow: -d.radius * 4, Life: 0.5, Max: 0.5, Bright: 3})
	g.addPart(Particle{Pos: pos, Col: d.col, Size: 2, Grow: d.radius * 8, Life: 0.5, Max: 0.5, Bright: 2, Shape: 1})
}

func (g *Game) addPart(p Particle) {
	if len(g.Parts) < maxParticles {
		g.Parts = append(g.Parts, p)
	}
}

func (g *Game) explode(pos V3, scale float32, col V3) {
	sfxPos(sfxExplode, float32(math.Min(1.2, float64(0.55+0.3*scale))), pos)
	g.addPart(Particle{Pos: pos, Col: V3{1, 0.95, 0.85}, Size: 26 * scale, Life: 0.16, Max: 0.16, Bright: 6})
	g.addPart(Particle{Pos: pos, Col: col, Size: 3, Grow: 150 * scale, Life: 0.55, Max: 0.55, Bright: 2.5, Shape: 1})
	for i := 0; i < int(14*scale)+6; i++ {
		l := 0.45 + rnd()*0.5
		g.addPart(Particle{
			Pos: pos.Add(rndDir().Mul(3 * scale)), Vel: rndDir().Mul(rnd() * 30 * scale),
			Col: V3{1.0, 0.45 + rnd()*0.3, 0.12}, Size: (6 + rnd()*6) * scale, Grow: 6 * scale,
			Life: l, Max: l, Drag: 1.5, Bright: 2.2,
		})
	}
	for i := 0; i < int(34*scale)+10; i++ {
		l := 0.4 + rnd()*0.8
		c := col
		if rnd() < 0.4 {
			c = V3{1, 0.9, 0.7}
		}
		g.addPart(Particle{
			Pos: pos, Vel: rndDir().Mul((50 + rnd()*150) * scale), Col: c,
			Size: 1.0 + rnd()*1.4, Life: l, Max: l, Drag: 1.2, Bright: 4,
		})
	}
}

func (g *Game) sparks(pos V3, col V3, n int) {
	g.addPart(Particle{Pos: pos, Col: col, Size: 7, Life: 0.12, Max: 0.12, Bright: 4})
	for i := 0; i < n; i++ {
		l := 0.15 + rnd()*0.3
		g.addPart(Particle{Pos: pos, Vel: rndDir().Mul(30 + rnd()*90), Col: col, Size: 0.8 + rnd(), Life: l, Max: l, Drag: 2, Bright: 4})
	}
}

func (g *Game) hurtPlayer() {
	p := &g.P
	if p.Invuln > 0 || p.FlipT > 0 || g.State != stateRunning {
		return
	}
	p.Shields--
	p.Invuln = 1.2
	sfx(sfxHurt, 1)
	g.Shake, g.HitFx = 1, 1
	g.Combo, g.ComboT = 0, 0
	g.addPart(Particle{Pos: p.Pos, Col: V3{0.3, 0.7, 1}, Size: 4, Grow: 90, Life: 0.35, Max: 0.35, Bright: 3, Shape: 1})
	if p.Shields <= 0 {
		g.explode(p.Pos, 2.2, V3{0.4, 0.8, 1})
		g.State = stateGameOver
		g.Shake = 1.6
		sfx(sfxGameOver, 1)
		if g.Score > g.HiScore {
			g.HiScore, g.NewHi = g.Score, true
			saveHiScore(g.Score)
		}
	}
}

func (g *Game) killEnemy(e *Enemy) {
	d := kinds[e.Kind]
	if g.ComboT > 0 {
		g.Combo = min(g.Combo+1, 8)
	} else {
		g.Combo = 1
	}
	g.ComboT = 3
	g.Score += d.score * g.Combo * diffs[g.Diff].scoreX2 / 2
	g.Kills++
	sc := d.radius / 9
	g.explode(e.Pos, sc, d.col)
	g.Shake = float32(math.Max(float64(g.Shake), float64(0.15*sc)))
	drops, chance := 1, float32(0.13)
	switch e.Kind {
	case kBruiser:
		chance = 0.45
	case kBoss:
		drops, chance = 3, 1
		g.banner("DREADNOUGHT DESTROYED", "", 3)
	}
	for i := 0; i < drops; i++ {
		if rnd() < chance {
			kind := rand.Intn(4)
			if g.P.Shields <= 2 && rnd() < 0.5 {
				kind = puShield
			}
			g.Pickups = append(g.Pickups, Pickup{Pos: e.Pos.Add(rndDir().Mul(float32(i) * 14)), Kind: kind, Life: 22})
		}
	}
}

func (g *Game) enemyFire(e *Enemy, spread int, gap float32) {
	const speed = 190
	p := &g.P
	dist := p.Pos.Sub(e.Pos).Len()
	target := p.Pos.Add(p.Vel.Mul(dist / speed * 0.75)).Add(rndDir().Mul(7))
	dir := norm(target.Sub(e.Pos))
	side := norm(dir.Cross(V3{0, 1, 0}))
	col := kinds[e.Kind].col
	sfxPos(sfxEnemyShot, 1, e.Pos)
	for i := 0; i < spread; i++ {
		o := (float32(i) - float32(spread-1)/2) * gap
		d := norm(dir.Add(side.Mul(o)))
		pos := e.Pos.Add(d.Mul(kinds[e.Kind].radius * 0.8))
		g.Shots = append(g.Shots, Shot{Pos: pos, Prev: pos, Vel: d.Mul(speed), Life: 5, Hostile: true, Col: col})
	}
	g.addPart(Particle{Pos: e.Pos.Add(dir.Mul(kinds[e.Kind].radius * 0.8)), Col: col, Size: 10, Life: 0.1, Max: 0.1, Bright: 4})
}

func approach(cur, target V3, rate float32) V3 {
	if rate > 1 {
		rate = 1
	}
	return cur.Add(target.Sub(cur).Mul(rate))
}

func (g *Game) Update(dt float32, in Input) {
	g.Time += dt
	g.Shake *= float32(math.Exp(float64(-6 * dt)))
	g.HitFx = float32(math.Max(0, float64(g.HitFx-dt*1.6)))
	g.HitMark -= dt
	g.BannerT -= dt
	g.NoteT -= dt

	if g.State == stateMenu || g.State == stateName {
		g.P.Yaw += 5 * dt
		g.emitPlayerTrail(dt)
		g.updateAsteroids(dt)
		g.updateParticles(dt)
		return
	}
	if g.State == statePaused {
		return
	}
	if g.State == stateRunning {
		g.updatePlayer(dt, in)
		g.updateWaves(dt)
	}
	g.updateEnemies(dt)
	g.updateShots(dt)
	g.updateRockets(dt)
	g.updatePickups(dt)
	g.updateAsteroids(dt)
	g.updateParticles(dt)
}

func (g *Game) Look(dx, dy float32) {
	if g.State != stateRunning {
		return
	}
	p := &g.P
	p.Yaw += dx
	p.Pitch = mgl.Clamp(p.Pitch-dy, -85, 85)
	p.yawRate += dx
	p.pitRate -= dy
}

func (g *Game) emitPlayerTrail(dt float32) {
	p := &g.P
	boosting := p.BoostT > 0
	rate := float32(110)
	if boosting {
		rate = 220
	}
	p.trailAcc += rate * dt
	m := p.Model()
	back := p.Front().Mul(-1)
	for ; p.trailAcc >= 1; p.trailAcc-- {
		for _, e := range playerEngines {
			pos := m.Mul4x1(e.Vec4(1)).Vec3()
			pt := Particle{Pos: pos, Vel: p.Vel.Add(back.Mul(35 + rnd()*20)).Add(rndDir().Mul(3)),
				Col: V3{0.25, 0.7, 1}, Size: 1.5, Grow: -3.5, Life: 0.30, Max: 0.30, Bright: 0.35}
			if boosting {
				pt.Col, pt.Size, pt.Life, pt.Max, pt.Grow, pt.Bright = V3{1, 0.6, 0.25}, 2.4, 0.45, 0.45, -4, 0.6
			}
			g.addPart(pt)
		}
	}
}

func (g *Game) updatePlayer(dt float32, in Input) {
	p := &g.P
	f, r := p.Front(), p.Right()

	// yawRate/pitRate accumulate mouse deltas; turn them into smoothed deg/s
	k := float32(math.Exp(float64(-10 * dt)))
	p.yawRate *= k
	p.pitRate *= k
	targetRoll := mgl.Clamp(-p.yawRate*0.09-in.Strafe*0.38, -1.0, 1.0)
	p.Roll += (targetRoll - p.Roll) * float32(math.Min(1, float64(7*dt)))

	// flip: barrel-roll sideways out of the line of fire; shots pass through while it lasts
	p.FlipCD -= dt
	if (in.FlipL || in.FlipR) && p.FlipCD <= 0 && p.FlipT <= 0 {
		p.flipDir = 1
		if in.FlipL {
			p.flipDir = -1
		}
		p.FlipT, p.FlipCD = flipTime, flipCD
		sfx(sfxBoost, 0.7)
	}
	p.FlipT -= dt

	p.BoostCD -= dt
	p.BoostT -= dt
	if in.Boost && p.BoostCD <= 0 {
		p.BoostT, p.BoostCD = boostTime, boostCD
		sfx(sfxBoost, 1)
		g.Shake = float32(math.Max(float64(g.Shake), 0.25))
		g.addPart(Particle{Pos: p.Pos.Sub(f.Mul(6)), Col: V3{1, 0.7, 0.3}, Size: 3, Grow: 110, Life: 0.3, Max: 0.3, Bright: 3, Shape: 1})
	}
	var want V3
	if p.BoostT > 0 {
		want = f.Mul(boostSpeed).Add(r.Mul(in.Strafe * playerSpeed * 0.5))
	} else {
		a := f.Mul(in.Fwd).Add(r.Mul(in.Strafe))
		if a.Len() > 0.01 {
			want = a.Normalize().Mul(playerSpeed)
		}
	}
	rate := float32(4.5)
	if p.FlipT > 0 {
		want = want.Add(r.Mul(p.flipDir * flipSpeed))
		rate = 14
	}
	p.Vel = approach(p.Vel, want, rate*dt)
	p.Pos = p.Pos.Add(p.Vel.Mul(dt))
	g.emitPlayerTrail(dt)

	// rockets
	g.Lock = g.pickLock()
	p.rocketCD -= dt
	if in.Rocket && p.rocketCD <= 0 && p.Rockets > 0 && p.FlipT <= 0 {
		g.fireRocket()
	}

	// weapons
	p.Invuln -= dt
	p.RapidT -= dt
	p.CoolT -= dt
	p.fireCD -= dt
	if p.Heat > 0 {
		p.Heat -= coolRate * dt
	}
	if p.Heat <= 0.15 {
		p.Overheated = false
	}
	if p.Heat < 0 {
		p.Heat = 0
	}
	if in.Fire && p.fireCD <= 0 && !p.Overheated {
		rate, heat := float32(fireRate), float32(heatPerShot)
		if p.RapidT > 0 {
			rate, heat = rate*0.55, heat*0.5
		}
		if p.CoolT > 0 {
			heat = 0
		}
		p.fireCD = rate
		p.Heat += heat
		if p.Heat >= 1 {
			p.Heat, p.Overheated = 1, true
			sfx(sfxOverheat, 1)
			g.Note, g.NoteT, g.NoteBad = "WEAPONS OVERHEATED", 1.5, true
		}
		aim := p.Pos.Add(f.Mul(600))
		m := p.Model()
		sfx(sfxLaser, 1)
		col := V3{0.2, 0.85, 1.0}
		if p.RapidT > 0 {
			col = V3{1.0, 0.6, 0.15}
		}
		for _, gp := range playerGuns {
			pos := m.Mul4x1(gp.Vec4(1)).Vec3()
			dir := norm(aim.Sub(pos))
			g.Shots = append(g.Shots, Shot{Pos: pos, Prev: pos, Vel: dir.Mul(laserSpeed).Add(p.Vel.Mul(0.5)), Life: 1.6, Col: col})
			g.addPart(Particle{Pos: pos.Add(dir.Mul(2)), Vel: p.Vel, Col: col, Size: 5, Life: 0.07, Max: 0.07, Bright: 5})
		}
	}
}

func (g *Game) updateWaves(dt float32) {
	if len(g.queue) == 0 && len(g.Enemies) == 0 {
		if g.waveT <= 0 {
			// wave just cleared
			g.waveT = 3.5
			if g.Wave > 0 {
				bonus := 100 * g.Wave * diffs[g.Diff].scoreX2 / 2
				g.Score += bonus
				g.P.Rockets = min(g.P.Rockets+2, maxRockets)
				name := g.Name
				if name == "" {
					name = "PILOT"
				}
				g.banner(cheers[rand.Intn(len(cheers))]+", "+name+"!",
					"WAVE "+strconv.Itoa(g.Wave)+" CLEARED - BONUS +"+strconv.Itoa(bonus)+" - ROCKETS +2", 3.2)
				sfx(sfxPickup, 1)
			}
		}
		g.waveT -= dt
		if g.waveT <= 0 {
			g.startWave()
		}
		return
	}
	g.spawnT -= dt
	if len(g.queue) > 0 && g.spawnT <= 0 && len(g.Enemies) < 16 {
		kind := g.queue[0]
		g.queue = g.queue[1:]
		g.spawnT = 0.55
		f := g.P.Front()
		dir := norm(f.Add(rndDir().Mul(0.95)))
		g.spawnEnemy(kind, g.P.Pos.Add(dir.Mul(600+rnd()*150)))
	}
}

func (g *Game) updateEnemies(dt float32) {
	p := &g.P
	alive := g.State == stateRunning
	keep := g.Enemies[:0]
	for _, e := range g.Enemies {
		d := kinds[e.Kind]
		e.Age += dt
		e.Flash -= dt
		toP := p.Pos.Sub(e.Pos)
		dist := toP.Len()
		dir := norm(toP)
		side := norm(dir.Cross(V3{0, 1, 0}))
		upv := norm(side.Cross(dir))
		speed := (d.speed + float32(min(g.Wave, 15))*1.5) * diffs[g.Diff].speed
		gap := diffs[g.Diff].fireGap
		var want V3
		face := dir
		switch e.Kind {
		case kDrone:
			t := float64(e.Age*2.2 + e.phase)
			weave := side.Mul(float32(math.Sin(t))).Add(upv.Mul(float32(math.Cos(t * 0.8))))
			wk := mgl.Clamp((dist-60)/200, 0, 0.7)
			want = norm(dir.Add(weave.Mul(wk))).Mul(speed)
			if e.Vel.Len() > 1 {
				face = norm(e.Vel)
			}
			e.fireT -= dt
			if alive && e.fireT <= 0 && dist > 90 && dist < 460 {
				g.enemyFire(e, 1, 0)
				e.fireT = (2.6 + rnd()*2) * gap
			}
		case kFighter:
			e.strafeT -= dt
			if e.strafeT <= 0 {
				e.strafeT = 1.2 + rnd()*1.8
				s := rndDir()
				e.strafe = norm(s.Sub(dir.Mul(s.Dot(dir))))
			}
			radial := mgl.Clamp((dist-210)/80, -1, 1)
			want = dir.Mul(radial * speed).Add(e.strafe.Mul(speed * 0.75))
			e.fireT -= dt
			if alive && e.fireT <= 0 && dist < 480 {
				g.enemyFire(e, 1, 0)
				e.fireT = (float32(math.Max(1.0, 2.6-float64(g.Wave)*0.08)) + rnd()*0.8) * gap
			}
		case kBruiser:
			want = dir.Mul(speed)
			e.fireT -= dt
			if alive && e.fireT <= 0 && dist < 520 {
				g.enemyFire(e, 3, 0.07)
				e.fireT = (2.8 + rnd()) * gap
			}
		case kBoss:
			radial := mgl.Clamp((dist-300)/100, -1, 1)
			tang := norm(side.Add(upv.Mul(float32(math.Sin(float64(e.Age) * 0.3)))))
			want = dir.Mul(radial * speed).Add(tang.Mul(speed * 0.8))
			rage := e.HP < e.MaxHP*0.5
			e.fireT -= dt
			if alive && e.fireT <= 0 && dist < 700 {
				if rage {
					g.enemyFire(e, 7, 0.06)
					e.fireT = 0.9 * gap
				} else {
					g.enemyFire(e, 5, 0.07)
					e.fireT = 1.4 * gap
				}
			}
			e.spawnT -= dt
			if alive && e.spawnT <= 0 && len(g.Enemies) < 14 {
				e.spawnT = 7
				if rage {
					e.spawnT = 4.5
				}
				for i := 0; i < 2; i++ {
					g.queue = append(g.queue, kDrone)
				}
			}
		}
		// on the hardest level everything bigger than a drone carries rockets
		if diffs[g.Diff].enemyRockets && alive && e.Kind != kDrone {
			e.rocketT -= dt
			if e.rocketT <= 0 && dist > 140 && dist < 750 {
				g.enemyRocket(e)
				e.rocketT = [...]float32{0, 9, 6.5, 3.5}[e.Kind] + rnd()*3
			}
		}
		if !alive {
			want = e.Fwd.Mul(speed * 0.5)
			face = e.Fwd
		}
		e.Vel = approach(e.Vel, want, 2.5*dt)
		e.Pos = e.Pos.Add(e.Vel.Mul(dt))
		e.Fwd = norm(approach(e.Fwd, face, 4*dt))

		// engine trail
		e.trailAcc += 45 * dt
		for ; e.trailAcc >= 1; e.trailAcc-- {
			g.addPart(Particle{Pos: e.Pos.Sub(e.Fwd.Mul(d.tail * d.scale)), Vel: e.Vel.Mul(0.3).Add(rndDir().Mul(2)),
				Col: d.col, Size: 1.3 * d.scale, Grow: -3, Life: 0.4, Max: 0.4, Bright: 0.8})
		}

		if alive && dist < d.radius+5 {
			g.hurtPlayer()
			if e.Kind == kBoss {
				p.Vel = p.Vel.Add(dir.Mul(-260))
			} else {
				e.HP = 0
				g.explode(e.Pos, d.radius/9, d.col)
				continue
			}
		}
		if e.HP <= 0 {
			g.killEnemy(e)
			continue
		}
		keep = append(keep, e)
	}
	g.Enemies = keep
}

// segDist returns the distance from point c to segment a-b.
func segDist(a, b, c V3) float32 {
	ab := b.Sub(a)
	l2 := ab.Dot(ab)
	if l2 < 1e-6 {
		return c.Sub(a).Len()
	}
	t := mgl.Clamp(c.Sub(a).Dot(ab)/l2, 0, 1)
	return c.Sub(a.Add(ab.Mul(t))).Len()
}

func (g *Game) updateShots(dt float32) {
	p := &g.P
	keep := g.Shots[:0]
	for _, s := range g.Shots {
		s.Prev = s.Pos
		s.Pos = s.Pos.Add(s.Vel.Mul(dt))
		s.Life -= dt
		hit := s.Life <= 0
		if !hit {
			for i := range g.Asteroids {
				a := &g.Asteroids[i]
				if segDist(s.Prev, s.Pos, a.Pos) < a.Scale*0.95 {
					g.sparks(s.Pos, V3{1, 0.8, 0.5}, 6)
					if !s.Hostile {
						a.HP--
						sfxPos(sfxHit, 0.7, s.Pos)
					}
					hit = true
					break
				}
			}
		}
		if !hit && s.Hostile {
			if g.State == stateRunning && segDist(s.Prev, s.Pos, p.Pos) < 5.5 {
				g.sparks(s.Pos, s.Col, 10)
				g.hurtPlayer()
				hit = true
			}
		} else if !hit {
			for i := range g.Rockets {
				r := &g.Rockets[i]
				if r.Hostile && r.Life > 0 && segDist(s.Prev, s.Pos, r.Pos) < 6 {
					r.Life = 0 // detonates on its next update
					g.Score += 5
					g.HitMark = 0.14
					hit = true
					break
				}
			}
			for _, e := range g.Enemies {
				if hit {
					break
				}
				if e.HP > 0 && segDist(s.Prev, s.Pos, e.Pos) < kinds[e.Kind].radius {
					e.HP--
					e.Flash = 0.1
					sfx(sfxHit, 1)
					g.HitMark = 0.14
					g.sparks(s.Pos, s.Col, 8)
					hit = true
					break
				}
			}
		}
		if !hit {
			keep = append(keep, s)
		}
	}
	g.Shots = keep
}

func (g *Game) updatePickups(dt float32) {
	p := &g.P
	keep := g.Pickups[:0]
	for _, pu := range g.Pickups {
		pu.Life -= dt
		to := p.Pos.Sub(pu.Pos)
		d := to.Len()
		if d < 190 {
			pu.Pos = pu.Pos.Add(norm(to).Mul((190 - d + 60) * 1.4 * dt))
		}
		if g.State == stateRunning && d < 15 {
			switch pu.Kind {
			case puShield:
				p.Shields = min(p.Shields+1, maxShields)
			case puCoolant:
				p.Heat, p.Overheated, p.CoolT = 0, false, 6
			case puRapid:
				p.RapidT = 8
			case puRocket:
				p.Rockets = maxRockets // a full reload
			}
			g.Score += 5
			sfx(sfxPickup, 1)
			g.Note, g.NoteT, g.NoteBad = puNames[pu.Kind], 2, false
			g.addPart(Particle{Pos: p.Pos, Col: puCols[pu.Kind], Size: 4, Grow: 120, Life: 0.4, Max: 0.4, Bright: 3, Shape: 1})
			continue
		}
		if pu.Life > 0 {
			keep = append(keep, pu)
		}
	}
	g.Pickups = keep
}

func (g *Game) updateAsteroids(dt float32) {
	p := &g.P
	g.breakRocks()
	for i := range g.Asteroids {
		a := &g.Asteroids[i]
		a.Pos = a.Pos.Add(a.Vel.Mul(dt))
		a.Ang += a.Spin * dt
		// keep the field wrapped around the player
		for k := 0; k < 3; k++ {
			if a.Pos[k]-p.Pos[k] > fieldHalf {
				a.Pos[k] -= 2 * fieldHalf
			} else if a.Pos[k]-p.Pos[k] < -fieldHalf {
				a.Pos[k] += 2 * fieldHalf
			}
		}
		if g.State == stateRunning {
			to := p.Pos.Sub(a.Pos)
			if to.Len() < a.Scale*0.9+4 {
				n := norm(to)
				p.Pos = a.Pos.Add(n.Mul(a.Scale*0.9 + 4.5))
				p.Vel = n.Mul(90)
				g.sparks(p.Pos.Sub(n.Mul(4)), V3{1, 0.8, 0.5}, 14)
				g.hurtPlayer()
			}
		}
	}
}

func (g *Game) updateParticles(dt float32) {
	for i := 0; i < len(g.Parts); {
		pt := &g.Parts[i]
		pt.Life -= dt
		if pt.Life <= 0 {
			g.Parts[i] = g.Parts[len(g.Parts)-1]
			g.Parts = g.Parts[:len(g.Parts)-1]
			continue
		}
		if pt.Drag > 0 {
			pt.Vel = pt.Vel.Mul(float32(math.Exp(float64(-pt.Drag * dt))))
		}
		pt.Pos = pt.Pos.Add(pt.Vel.Mul(dt))
		pt.Size += pt.Grow * dt
		if pt.Size < 0 {
			pt.Size = 0
		}
		i++
	}
}
