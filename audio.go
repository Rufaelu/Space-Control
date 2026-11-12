package main

import (
	"math"
	"math/rand"

	"github.com/veandco/go-sdl2/sdl"
)

// All sound is synthesized at runtime, except songs the player puts in the
// music folder (see music.go), which replace the built-in soundtrack.
// Output is stereo: one-shot effects and continuous engine loops are
// panned and faded by where their source is relative to the player.

const sampleRate = 44100

const (
	sfxLaser = iota
	sfxEnemyShot
	sfxHit
	sfxExplode
	sfxHurt
	sfxPickup
	sfxBoost
	sfxOverheat
	sfxWave
	sfxGameOver
	sfxUIMove
	sfxUISelect
	sfxRocket
	sfxAlarm
)

const (
	wSine = iota
	wSquare
	wSaw
	wNoise // low-passed noise; the frequency sweep is the filter cutoff
)

type voice struct {
	wave             int
	f0, f1           float32 // frequency sweep, start to end
	dur, t, delay    float32
	vol, phase, filt float32
	pan              float32
	lpc              float32 // low-pass coefficient for tonal voices, 0 = unfiltered
}

// engineDef is the timbre of one class of ship engine.
type engineDef struct {
	freq, ratio2 float32 // base pitch, and the second oscillator's pitch ratio
	lfoHz        float32
	vibrato, am  float32 // how much the LFO bends pitch / pulses volume
	cutoff, vol  float32
}

// One continuous engine sound per enemy class, plus incoming rockets.
const (
	loopDrone = iota
	loopFighter
	loopBruiser
	loopBoss
	loopRocket
	loopCount
)

var engineDefs = [loopCount]engineDef{
	loopDrone:   {172, 1.50, 9.0, 0.045, 0.15, 1700, 0.20}, // angry insect whine
	loopFighter: {118, 1.012, 0.7, 0.02, 0.10, 1150, 0.22}, // twin-engine howl
	loopBruiser: {61, 0.50, 5.0, 0.0, 0.50, 520, 0.34},     // heavy chugging throb
	loopBoss:    {35, 1.50, 1.3, 0.01, 0.45, 300, 0.55},    // deep reactor rumble
	loopRocket:  {310, 2.02, 31, 0.06, 0.35, 2600, 0.20},   // rasping motor
}

type engineLoop struct {
	gain, pan, pitch float32 // targets, set once per frame
	g, p, pt         float32 // smoothed values
	ph1, ph2, lfo    float32
	filt             float32
}

type Audio struct {
	dev    sdl.AudioDeviceID
	voices []voice
	buf    []byte
	Volume int // 0..10
	Engine int // 0..10, loudness of all ship engines

	// listener
	pos, right V3

	// own ship
	thrust, boost, shipOn float32 // targets
	th, bo, on            float32 // smoothed
	sph1, sph2, sfilt     float32
	nfilt                 float32

	loops [loopCount]engineLoop

	// music sequencer
	MusicOn   bool
	musicMode int // 0 calm (title), 1 combat, 2 boss fight
	musicDuck float32
	seqT      float32
	seqStep   int
	music     *MusicPlayer // songs from the music folder
}

var audio *Audio

// sfx plays a centred sound effect; safe to call when audio is unavailable.
func sfx(id int, vol float32) {
	if audio != nil {
		audio.play(id, vol, 0)
	}
}

// sfxPos plays a sound at a world position: quieter with distance, panned to its side.
func sfxPos(id int, vol float32, pos V3) {
	if audio == nil {
		return
	}
	rel := pos.Sub(audio.pos)
	d := rel.Len()
	pan := float32(0)
	if d > 1 {
		pan = rel.Dot(audio.right) / d * 0.85
	}
	audio.play(id, vol/(1+d/250), pan)
}

func NewAudio() *Audio {
	if sdl.InitSubSystem(sdl.INIT_AUDIO) != nil {
		return nil
	}
	want := sdl.AudioSpec{Freq: sampleRate, Format: sdl.AUDIO_S16LSB, Channels: 2, Samples: 1024}
	dev, err := sdl.OpenAudioDevice("", false, &want, nil, 0)
	if err != nil {
		return nil
	}
	sdl.PauseAudioDevice(dev, false)
	a := newSynth()
	a.dev = dev
	return a
}

// newSynth makes the synthesizer without an output device.
func newSynth() *Audio {
	a := &Audio{Volume: 7, Engine: 10, right: V3{1, 0, 0}, MusicOn: true, musicDuck: 1}
	for i := range a.loops {
		a.loops[i].pitch, a.loops[i].pt = 1, 1
	}
	a.music = NewMusicPlayer(musicDir())
	return a
}

// Track is the title of the song playing from the music folder, or "".
func (a *Audio) Track() string {
	if a.MusicOn && a.music.Active() {
		return a.music.Title
	}
	return ""
}

// NextTrack skips to the next song in the music folder.
func (a *Audio) NextTrack() {
	if a.music.Active() {
		a.music.Skip()
	}
}

// Listen updates the listener and every engine loop from the game state.
func (a *Audio) Listen(g *Game) {
	p := &g.P
	a.pos, a.right = p.Pos, p.Right()
	for i := range a.loops {
		a.loops[i].gain = 0
	}
	a.thrust, a.boost, a.shipOn = 0, 0, 0
	a.MusicOn = g.Music
	a.Engine = g.EngineVol
	a.musicMode, a.musicDuck = 0, 1
	switch g.State {
	case stateRunning:
		a.musicMode = 1
		for _, e := range g.Enemies {
			if e.Kind == kBoss {
				a.musicMode = 2
			}
		}
	case statePaused:
		a.musicMode, a.musicDuck = 1, 0.45
	}
	switch g.State {
	case stateMenu, stateName:
		a.shipOn = 0.5
		return
	case stateRunning:
	default:
		return // paused or destroyed: engines fall silent
	}
	a.shipOn = 1
	a.thrust = float32(math.Min(1, float64(p.Vel.Len()/playerSpeed)))
	if p.BoostT > 0 {
		a.boost = 1
	}

	// each enemy class is one loop: louder the more ships are near, panned to
	// their side, and pitched up or down as they close in or pull away
	var wsum, pan, pitch [loopCount]float32
	feed := func(i int, pos, vel V3, reach float32) {
		rel := pos.Sub(p.Pos)
		d := rel.Len()
		if d < 1 {
			d = 1
		}
		dir := rel.Mul(1 / d)
		w := 1 / (1 + (d/reach)*(d/reach))
		closing := -vel.Sub(p.Vel).Dot(dir)
		wsum[i] += w
		pan[i] += w * dir.Dot(a.right)
		pitch[i] += w * (1 + mglClamp(closing/450, -0.22, 0.35))
	}
	for _, e := range g.Enemies {
		reach := float32(130)
		if e.Kind == kBoss {
			reach = 380
		}
		feed(e.Kind, e.Pos, e.Vel, reach)
	}
	for i := range g.Rockets {
		if r := &g.Rockets[i]; r.Hostile {
			feed(loopRocket, r.Pos, r.Dir.Mul(r.Speed), 160)
		}
	}
	for i := range a.loops {
		if wsum[i] > 0 {
			l := &a.loops[i]
			l.gain = float32(math.Min(1.1, float64(wsum[i])))
			l.pan = pan[i] / wsum[i] * 0.85
			l.pitch = pitch[i] / wsum[i]
		}
	}
}

func mglClamp(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (a *Audio) add(wave int, f0, f1, dur, vol, delay, pan float32) {
	if len(a.voices) >= 72 {
		return
	}
	a.voices = append(a.voices, voice{wave: wave, f0: f0, f1: f1, dur: dur, vol: vol, delay: delay, pan: pan})
}

func (a *Audio) play(id int, v, pan float32) {
	add := func(wave int, f0, f1, dur, vol, delay float32) { a.add(wave, f0, f1, dur, vol*v, delay, pan) }
	switch id {
	case sfxLaser:
		p := 0.92 + rand.Float32()*0.16
		add(wSaw, 1500*p, 260*p, 0.11, 0.13, 0)
		add(wSine, 2400*p, 500*p, 0.07, 0.09, 0)
		add(wNoise, 6000, 2000, 0.04, 0.08, 0)
	case sfxEnemyShot:
		add(wSquare, 520, 170, 0.16, 0.12, 0)
		add(wNoise, 3000, 800, 0.08, 0.10, 0)
	case sfxHit:
		add(wNoise, 5000, 1500, 0.05, 0.22, 0)
		add(wSine, 900, 600, 0.05, 0.10, 0)
	case sfxExplode:
		add(wNoise, 2600, 110, 0.85, 0.80, 0)
		add(wSine, 130, 32, 0.60, 0.75, 0)
		add(wNoise, 7000, 900, 0.12, 0.35, 0)
		add(wNoise, 500, 80, 1.30, 0.35, 0.05)
	case sfxHurt:
		add(wSaw, 240, 55, 0.40, 0.40, 0)
		add(wNoise, 3500, 300, 0.30, 0.45, 0)
	case sfxPickup:
		add(wSine, 660, 660, 0.10, 0.22, 0)
		add(wSine, 880, 880, 0.10, 0.22, 0.08)
		add(wSine, 1320, 1320, 0.22, 0.22, 0.16)
	case sfxBoost:
		add(wNoise, 300, 5000, 0.55, 0.40, 0)
		add(wSaw, 70, 240, 0.55, 0.18, 0)
	case sfxOverheat:
		add(wSquare, 230, 230, 0.12, 0.16, 0)
		add(wSquare, 230, 230, 0.12, 0.16, 0.18)
		add(wSquare, 170, 170, 0.22, 0.16, 0.36)
	case sfxWave:
		add(wSine, 330, 330, 0.16, 0.18, 0)
		add(wSine, 440, 440, 0.16, 0.18, 0.13)
		add(wSine, 660, 660, 0.40, 0.20, 0.26)
	case sfxGameOver:
		add(wSaw, 330, 300, 0.30, 0.20, 0.5)
		add(wSaw, 247, 225, 0.30, 0.20, 0.8)
		add(wSaw, 165, 70, 0.90, 0.24, 1.1)
	case sfxUIMove:
		add(wSine, 700, 700, 0.04, 0.15, 0)
	case sfxUISelect:
		add(wSine, 700, 700, 0.05, 0.18, 0)
		add(wSine, 1050, 1050, 0.09, 0.18, 0.05)
	case sfxRocket:
		add(wNoise, 900, 4500, 0.70, 0.45, 0)
		add(wSaw, 110, 320, 0.60, 0.22, 0)
	case sfxAlarm:
		add(wSquare, 990, 990, 0.09, 0.12, 0)
		add(wSquare, 740, 740, 0.09, 0.12, 0.11)
	}
}

func wrap(ph float32) float32 {
	if ph >= 1 {
		ph -= float32(int(ph))
	}
	return ph
}

func lpCoef(cutoff float32) float32 {
	return 1 - float32(math.Exp(float64(-2*math.Pi*cutoff/sampleRate)))
}

// Pump keeps roughly 70 ms of audio queued; call once per frame.
func (a *Audio) Pump() {
	const target = sampleRate * 70 / 1000
	queued := int(sdl.GetQueuedAudioSize(a.dev)) / 4
	n := target - queued
	if n <= 0 {
		return
	}
	if n > 4096 {
		n = 4096
	}
	_ = sdl.QueueAudio(a.dev, a.render(n))
}

// render synthesizes n stereo frames of 16-bit audio.
func (a *Audio) render(n int) []byte {
	if cap(a.buf) < n*4 {
		a.buf = make([]byte, n*4)
	}
	buf := a.buf[:n*4]
	const dt = 1.0 / sampleRate
	const smooth = 0.0006 // per-sample glide toward targets, about 40 ms
	master := float32(a.Volume) / 10
	master *= master
	engine := float32(a.Engine) / 10
	engine *= engine

	var coef [loopCount]float32
	for i, d := range engineDefs {
		coef[i] = lpCoef(d.cutoff)
	}
	custom := a.music.Active()

	for i := 0; i < n; i++ {
		var left, right float32
		mixIn := func(s, pan float32) {
			left += s * float32(math.Min(1, float64(1-pan)))
			right += s * float32(math.Min(1, float64(1+pan)))
		}

		// --- music ---
		a.seqT += dt
		if a.seqT >= stepLen {
			a.seqT -= stepLen
			if a.MusicOn && !custom {
				a.musicStep()
			}
			a.seqStep = (a.seqStep + 1) % (64 * 2)
		}
		if custom && a.MusicOn {
			if l, r, ok := a.music.next(); ok {
				left += l * musicGain * a.musicDuck
				right += r * musicGain * a.musicDuck
			}
		}

		// --- own engine: a low twin-oscillator drone plus thrust hiss ---
		a.th += (a.thrust - a.th) * smooth * 0.5
		a.bo += (a.boost - a.bo) * smooth
		a.on += (a.shipOn - a.on) * smooth
		if a.on > 0.001 {
			f := 46 + 44*a.th + 60*a.bo
			a.sph1 = wrap(a.sph1 + f*dt)
			a.sph2 = wrap(a.sph2 + f*2.013*dt)
			raw := (a.sph1*2 - 1) + 0.6*(a.sph2*2-1)
			a.sfilt += (raw - a.sfilt) * lpCoef(240+700*a.th+1500*a.bo)
			a.nfilt += (rand.Float32()*2 - 1 - a.nfilt) * lpCoef(500+1500*a.th+4000*a.bo)
			s := a.sfilt*(0.16+0.10*a.th+0.12*a.bo) + a.nfilt*(0.05+0.18*a.th+0.55*a.bo)
			mixIn(s*a.on*engine, 0)
		}

		// --- other ships ---
		for k := range a.loops {
			l := &a.loops[k]
			l.g += (l.gain - l.g) * smooth
			if l.g < 0.0008 {
				continue
			}
			l.p += (l.pan - l.p) * smooth
			l.pt += (l.pitch - l.pt) * smooth
			d := &engineDefs[k]
			l.lfo = wrap(l.lfo + d.lfoHz*dt)
			lfo := float32(math.Sin(float64(l.lfo) * 2 * math.Pi))
			f := d.freq * l.pt * (1 + d.vibrato*lfo)
			l.ph1 = wrap(l.ph1 + f*dt)
			l.ph2 = wrap(l.ph2 + f*d.ratio2*dt)
			raw := (l.ph1*2 - 1) + 0.7*(l.ph2*2-1)
			if k == loopBoss || k == loopRocket {
				raw += (rand.Float32()*2 - 1) * 1.2
			}
			l.filt += (raw - l.filt) * coef[k]
			mixIn(l.filt*(1+d.am*lfo)*d.vol*l.g*engine, l.p)
		}

		// --- one-shot effects ---
		for j := range a.voices {
			v := &a.voices[j]
			if v.delay > 0 {
				v.delay -= dt
				continue
			}
			if v.t >= v.dur {
				continue
			}
			k := v.t / v.dur
			f := v.f0 * float32(math.Pow(float64(v.f1/v.f0), float64(k)))
			var s float32
			switch v.wave {
			case wSine:
				s = float32(math.Sin(float64(v.phase) * 2 * math.Pi))
			case wSquare:
				s = 0.7
				if v.phase > 0.5 {
					s = -0.7
				}
			case wSaw:
				s = (v.phase*2 - 1) * 0.8
			case wNoise:
				v.filt += (rand.Float32()*2 - 1 - v.filt) * lpCoef(f)
				s = v.filt * 1.6
			}
			if v.lpc > 0 {
				v.filt += (s - v.filt) * v.lpc
				s = v.filt
			}
			v.phase = wrap(v.phase + f*dt)
			env := (1 - k) * (1 - k)
			if v.t < 0.004 {
				env *= v.t / 0.004
			}
			mixIn(s*env*v.vol, v.pan)
			v.t += dt
		}

		for c, s := range [2]float32{left, right} {
			s16 := int16(float32(math.Tanh(float64(s*master*1.3))) * 30000)
			buf[i*4+c*2] = byte(s16)
			buf[i*4+c*2+1] = byte(uint16(s16) >> 8)
		}
	}
	keep := a.voices[:0]
	for _, v := range a.voices {
		if v.t < v.dur {
			keep = append(keep, v)
		}
	}
	a.voices = keep
	return buf
}

// --- music: an original 138 BPM synth loop, sequenced in sixteenth notes ---

const stepLen = 60.0 / 138 / 4

func midi(n int) float32 { return 440 * float32(math.Pow(2, float64(n-69)/12)) }

// Four bars: A minor, F, C, G. Each entry is bass root and chord tones (MIDI notes).
var chords = [4]struct {
	root  int
	tones [4]int
}{
	{33, [4]int{57, 60, 64, 69}}, // Am
	{29, [4]int{57, 60, 65, 69}}, // F
	{36, [4]int{55, 60, 64, 67}}, // C
	{31, [4]int{55, 59, 62, 67}}, // G
}

// Lead melody over the second pass of the loop: step, MIDI note, length in steps.
var lead = [...][3]int{
	{0, 76, 6}, {6, 74, 2}, {8, 72, 6}, {14, 71, 2},
	{16, 69, 8}, {24, 72, 4}, {28, 76, 4},
	{32, 79, 6}, {38, 76, 6}, {44, 74, 4},
	{48, 71, 6}, {54, 74, 4}, {58, 79, 6},
}

var (
	bassOct = [16]int{0, 0, 12, 0, 0, 12, 0, 0, 0, 0, 12, 0, 0, 12, 0, 12}
	arpIdx  = [16]int{0, 1, 2, 3, 2, 1, 2, 3, 0, 1, 2, 3, 2, 3, 1, 2}
)

// note adds one music voice; cutoff 0 leaves it unfiltered.
func (a *Audio) note(wave int, f0, f1, dur, vol, cutoff, pan float32) {
	a.add(wave, f0, f1, dur, vol*0.8*a.musicDuck, 0, pan)
	if cutoff > 0 && len(a.voices) > 0 {
		a.voices[len(a.voices)-1].lpc = lpCoef(cutoff)
	}
}

func (a *Audio) musicStep() {
	step := a.seqStep % 64
	pass := a.seqStep / 64 // the loop alternates: without and with the lead
	s := step % 16
	ch := chords[step/16]
	mode := a.musicMode

	// drums
	if s%4 == 0 {
		kv := float32(0.55)
		if mode == 0 {
			kv = 0.28
		}
		a.note(wSine, 155, 42, 0.20, kv, 0, 0)
		a.note(wNoise, 4000, 600, 0.02, kv*0.4, 0, 0)
	}
	if mode > 0 {
		if s == 4 || s == 12 {
			a.note(wNoise, 3200, 1400, 0.15, 0.30, 0, 0)
			a.note(wSine, 200, 150, 0.08, 0.20, 0, 0)
		}
		if s%4 == 2 {
			a.note(wNoise, 11000, 9000, 0.05, 0.11, 0, 0.2)
		} else if mode == 2 || s%2 == 1 {
			a.note(wNoise, 11000, 9000, 0.02, 0.05, 0, -0.2)
		}
		if step == 62 || step == 63 || (mode == 2 && s == 15) {
			a.note(wNoise, 3200, 1600, 0.09, 0.20, 0, 0) // snare fill into the next loop
		}
	}

	// bass: pulsing root with octave jumps
	bf := midi(ch.root + bassOct[s])
	bv := float32(0.30)
	if mode == 0 {
		bv = 0.18
		if s%2 == 1 {
			bv = 0
		}
	}
	if bv > 0 {
		a.note(wSaw, bf, bf, 0.16, bv, 420, 0)
	}

	// arpeggio
	af := midi(ch.tones[arpIdx[s]] + 12)
	pan := float32(0.45)
	if s%2 == 1 {
		pan = -0.45
	}
	a.note(wSquare, af, af, 0.16, 0.085, 2400, pan)

	// pad: the chord, once a bar
	if s == 0 {
		for _, t := range ch.tones[:3] {
			f := midi(t)
			a.note(wSine, f, f, 2.4, 0.075, 0, 0)
		}
	}

	// lead
	if mode > 0 && pass == 1 {
		for _, n := range lead {
			if n[0] == step {
				f := midi(n[1])
				d := float32(n[2])*stepLen*1.6 + 0.1
				a.note(wSaw, f, f, d, 0.11, 2000, 0.15)
				a.note(wSquare, f*2.005, f*2.005, d, 0.035, 2600, -0.15)
			}
		}
	}
}
