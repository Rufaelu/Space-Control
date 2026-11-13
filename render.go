package main

import (
	"fmt"
	"math"
	"math/rand"
	"strings"

	"game/internal/gl"
	mgl "github.com/go-gl/mathgl/mgl32"
)

var (
	sunDir    = norm(V3{-0.55, 0.32, -0.77})
	planetDir = norm(V3{0.62, -0.12, -0.78})
)

// --- shader program wrapper ---

type Prog struct {
	id   uint32
	locs map[string]int32
}

func compileShader(src string, typ uint32) uint32 {
	s := gl.CreateShader(typ)
	cstr, free := gl.Strs(src)
	gl.ShaderSource(s, 1, cstr, nil)
	free()
	gl.CompileShader(s)
	var ok int32
	gl.GetShaderiv(s, gl.COMPILE_STATUS, &ok)
	if ok == gl.FALSE {
		var n int32
		gl.GetShaderiv(s, gl.INFO_LOG_LENGTH, &n)
		log := strings.Repeat("\x00", int(n+1))
		gl.GetShaderInfoLog(s, n, nil, gl.Str(log))
		panic(fmt.Sprintf("shader compile failed: %s\n%s", log, src))
	}
	return s
}

func newProg(vs, fs string) *Prog {
	v, f := compileShader(vs, gl.VERTEX_SHADER), compileShader(fs, gl.FRAGMENT_SHADER)
	p := gl.CreateProgram()
	gl.AttachShader(p, v)
	gl.AttachShader(p, f)
	gl.LinkProgram(p)
	var ok int32
	gl.GetProgramiv(p, gl.LINK_STATUS, &ok)
	if ok == gl.FALSE {
		var n int32
		gl.GetProgramiv(p, gl.INFO_LOG_LENGTH, &n)
		log := strings.Repeat("\x00", int(n+1))
		gl.GetProgramInfoLog(p, n, nil, gl.Str(log))
		panic("shader link failed: " + log)
	}
	gl.DeleteShader(v)
	gl.DeleteShader(f)
	return &Prog{id: p, locs: map[string]int32{}}
}

func (p *Prog) use() { gl.UseProgram(p.id) }
func (p *Prog) loc(n string) int32 {
	l, ok := p.locs[n]
	if !ok {
		l = gl.GetUniformLocation(p.id, gl.Str(n+"\x00"))
		p.locs[n] = l
	}
	return l
}
func (p *Prog) i1(n string, v int32)            { gl.Uniform1i(p.loc(n), v) }
func (p *Prog) f1(n string, v float32)          { gl.Uniform1f(p.loc(n), v) }
func (p *Prog) f2(n string, a, b float32)       { gl.Uniform2f(p.loc(n), a, b) }
func (p *Prog) v3(n string, v V3)               { gl.Uniform3f(p.loc(n), v[0], v[1], v[2]) }
func (p *Prog) f4(n string, a, b, c, d float32) { gl.Uniform4f(p.loc(n), a, b, c, d) }
func (p *Prog) m4(n string, m mgl.Mat4)         { gl.UniformMatrix4fv(p.loc(n), 1, false, &m[0]) }

// --- GPU resources ---

type Mesh struct {
	vao uint32
	n   int32
}

func newMesh(v []float32) Mesh {
	var vao, vbo uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(v)*4, gl.Ptr(v), gl.STATIC_DRAW)
	attribs(10, 3, 3, 4)
	return Mesh{vao, int32(len(v) / 10)}
}

// attribs declares consecutive float attributes of the given sizes.
func attribs(stride int32, sizes ...int32) {
	off := 0
	for i, s := range sizes {
		gl.EnableVertexAttribArray(uint32(i))
		gl.VertexAttribPointer(uint32(i), s, gl.FLOAT, false, stride*4, gl.PtrOffset(off*4))
		off += int(s)
	}
}

// Batch is a per-frame dynamic vertex buffer.
type Batch struct {
	vao, vbo uint32
	stride   int
	data     []float32
}

func newBatch(stride int32, sizes ...int32) *Batch {
	b := &Batch{stride: int(stride)}
	gl.GenVertexArrays(1, &b.vao)
	gl.BindVertexArray(b.vao)
	gl.GenBuffers(1, &b.vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, b.vbo)
	attribs(stride, sizes...)
	return b
}

func (b *Batch) flush() {
	if len(b.data) == 0 {
		return
	}
	gl.BindVertexArray(b.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, b.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(b.data)*4, gl.Ptr(b.data), gl.STREAM_DRAW)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(b.data)/b.stride))
	b.data = b.data[:0]
}

type Target struct {
	fbo, tex uint32
	w, h     int32
}

// colorFormat is an internal texture format and the pixel type to upload it with.
type colorFormat struct{ internal, typ uint32 }

var (
	fmtHDR = colorFormat{gl.RGBA16F, gl.FLOAT}
	fmtLDR = colorFormat{gl.RGBA8, gl.UNSIGNED_BYTE}
)

func newTarget(w, h int32, f colorFormat) Target {
	t := Target{w: w, h: h}
	gl.GenTextures(1, &t.tex)
	gl.BindTexture(gl.TEXTURE_2D, t.tex)
	gl.TexImage2D(gl.TEXTURE_2D, 0, int32(f.internal), w, h, 0, gl.RGBA, f.typ, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.GenFramebuffers(1, &t.fbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, t.fbo)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, t.tex, 0)
	return t
}

func (t *Target) free() {
	gl.DeleteFramebuffers(1, &t.fbo)
	gl.DeleteTextures(1, &t.tex)
}

func (t *Target) bind() {
	gl.BindFramebuffer(gl.FRAMEBUFFER, t.fbo)
	gl.Viewport(0, 0, t.w, t.h)
}

type Renderer struct {
	w, h       int32 // the 3D scene's resolution
	outW, outH int32 // the window's; the HUD is drawn at this resolution
	maxH       int32 // cap on the scene's height (phones), 0 = none

	color   colorFormat // scene and bloom targets: HDR where the GPU can render it
	samples int32       // multisampling of the scene

	sky, star, mesh, part, bright, blur, comp, hud *Prog

	triVAO  uint32
	skyTex  uint32
	starVAO uint32
	starN   int32

	msFBO, msColor, msDepth uint32
	scene                   Target
	half, quarter           [2]Target
	final                   Target // only used for screenshots
	toFinal                 bool

	mPlayer, mPickup, mPlanet Mesh
	mRocket                   Mesh
	mEnemy                    [4]Mesh
	mAsteroid                 [4]Mesh

	parts *Batch
	hudB  *Batch

	// camera of the current frame
	camPos, camF, camR, camU V3
	vp                       mgl.Mat4
	fov                      float32
	cockpit                  bool // this frame is drawn from inside the ship
}

func NewRenderer(w, h int32, toFinal bool) *Renderer {
	r := &Renderer{toFinal: toFinal, fov: 0.95}
	r.sky = newProg(fsTriVS, skyFS)
	r.star = newProg(starVS, starFS)
	r.mesh = newProg(meshVS, meshFS)
	r.part = newProg(partVS, partFS)
	r.bright = newProg(fsTriVS, brightFS)
	r.blur = newProg(fsTriVS, blurFS)
	r.comp = newProg(fsTriVS, compositeFS)
	r.hud = newProg(hudVS, hudFS)

	gl.GenVertexArrays(1, &r.triVAO)
	if !gl.IsES { // always on in OpenGL ES
		gl.Enable(gl.TEXTURE_CUBE_MAP_SEAMLESS)
		gl.Enable(gl.PROGRAM_POINT_SIZE)
	} else {
		r.maxH = 720 // phones: render the 3D at up to 720 lines and scale it up
	}
	r.pickFormat()

	r.bakeSky()
	r.buildStars()

	r.mPlayer = newMesh(buildPlayer())
	r.mEnemy = [4]Mesh{newMesh(buildDrone()), newMesh(buildFighter()), newMesh(buildBruiser()), newMesh(buildBoss())}
	r.mPickup = newMesh(buildPickup())
	r.mRocket = newMesh(buildRocket())
	r.mPlanet = newMesh(buildPlanet())
	for i := range r.mAsteroid {
		r.mAsteroid[i] = newMesh(buildAsteroid(float32(i + 1)))
	}
	r.parts = newBatch(10, 3, 3, 4)
	r.hudB = newBatch(6, 2, 4)
	r.resize(w, h)
	return r
}

// pickFormat chooses HDR colour targets when the GPU can render (and
// multisample) them. Some phone GPUs cannot; they get plain 8-bit colour,
// which loses the bloom but still draws everything.
func (r *Renderer) pickFormat() {
	var maxSamples int32
	gl.GetIntegerv(gl.MAX_SAMPLES, &maxSamples)
	want := min(4, maxSamples)
	for _, c := range []struct {
		f colorFormat
		n int32
	}{{fmtHDR, want}, {fmtHDR, 0}, {fmtLDR, want}, {fmtLDR, 0}} {
		if targetWorks(c.f, c.n) {
			r.color, r.samples = c.f, c.n
			return
		}
	}
	r.color, r.samples = fmtLDR, 0
}

// targetWorks tries a small colour texture and a multisampled renderbuffer of a format.
func targetWorks(f colorFormat, samples int32) bool {
	t := newTarget(8, 8, f)
	ok := gl.CheckFramebufferStatus(gl.FRAMEBUFFER) == gl.FRAMEBUFFER_COMPLETE
	t.free()
	if ok && samples > 0 {
		var fbo, rb uint32
		gl.GenFramebuffers(1, &fbo)
		gl.BindFramebuffer(gl.FRAMEBUFFER, fbo)
		gl.GenRenderbuffers(1, &rb)
		gl.BindRenderbuffer(gl.RENDERBUFFER, rb)
		gl.RenderbufferStorageMultisample(gl.RENDERBUFFER, samples, f.internal, 8, 8)
		gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.RENDERBUFFER, rb)
		ok = gl.CheckFramebufferStatus(gl.FRAMEBUFFER) == gl.FRAMEBUFFER_COMPLETE
		gl.DeleteRenderbuffers(1, &rb)
		gl.DeleteFramebuffers(1, &fbo)
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	return ok
}

func (r *Renderer) bakeSky() {
	size := int32(1024)
	if gl.IsES {
		size = 512 // baking runs on the phone at start-up; keep it quick
	}
	gl.GenTextures(1, &r.skyTex)
	gl.BindTexture(gl.TEXTURE_CUBE_MAP, r.skyTex)
	for i := uint32(0); i < 6; i++ {
		gl.TexImage2D(gl.TEXTURE_CUBE_MAP_POSITIVE_X+i, 0, int32(r.color.internal), size, size, 0, gl.RGBA, r.color.typ, nil)
	}
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_WRAP_R, gl.CLAMP_TO_EDGE)

	bake := newProg(fsTriVS, bakeFS)
	bake.use()
	bake.v3("uSun", sunDir)
	bake.f1("uSeed", rand.Float32()*40)
	// forward, right, up of each cube face in GL's face layout
	faces := [6][3]V3{
		{{1, 0, 0}, {0, 0, -1}, {0, -1, 0}},
		{{-1, 0, 0}, {0, 0, 1}, {0, -1, 0}},
		{{0, 1, 0}, {1, 0, 0}, {0, 0, 1}},
		{{0, -1, 0}, {1, 0, 0}, {0, 0, -1}},
		{{0, 0, 1}, {1, 0, 0}, {0, -1, 0}},
		{{0, 0, -1}, {-1, 0, 0}, {0, -1, 0}},
	}
	var fbo uint32
	gl.GenFramebuffers(1, &fbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, fbo)
	gl.Viewport(0, 0, size, size)
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.BLEND)
	gl.BindVertexArray(r.triVAO)
	for i, f := range faces {
		gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_CUBE_MAP_POSITIVE_X+uint32(i), r.skyTex, 0)
		bake.v3("uF", f[0])
		bake.v3("uR", f[1])
		bake.v3("uU", f[2])
		gl.DrawArrays(gl.TRIANGLES, 0, 3)
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.DeleteFramebuffers(1, &fbo)
	gl.DeleteProgram(bake.id)
}

func (r *Renderer) buildStars() {
	const n = 2800
	palette := []V3{{0.65, 0.78, 1.0}, {0.85, 0.90, 1.0}, {1, 1, 1}, {1.0, 0.93, 0.78}, {1.0, 0.78, 0.55}, {1.0, 0.6, 0.45}}
	data := make([]float32, 0, n*8)
	for i := 0; i < n; i++ {
		d := rndDir()
		c := palette[rand.Intn(len(palette))]
		m := rnd()
		m = m * m * m
		bright := 0.5 + m*3.5
		size := 1.6 + m*4.5
		data = append(data, d[0], d[1], d[2], c[0]*bright, c[1]*bright, c[2]*bright, size, rnd())
	}
	var vbo uint32
	gl.GenVertexArrays(1, &r.starVAO)
	gl.BindVertexArray(r.starVAO)
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(data)*4, gl.Ptr(data), gl.STATIC_DRAW)
	attribs(8, 3, 4, 1)
	r.starN = n
}

// resize follows the window's drawable size; the scene may render smaller (maxH).
func (r *Renderer) resize(outW, outH int32) {
	if outW < 2 || outH < 2 || (outW == r.outW && outH == r.outH) {
		return
	}
	w, h := outW, outH
	if r.maxH > 0 && h > r.maxH {
		w, h = int32(int64(w)*int64(r.maxH)/int64(h)), r.maxH
	}
	r.outW, r.outH = outW, outH
	if r.w != 0 {
		gl.DeleteFramebuffers(1, &r.msFBO)
		gl.DeleteRenderbuffers(1, &r.msColor)
		gl.DeleteRenderbuffers(1, &r.msDepth)
		r.scene.free()
		for i := 0; i < 2; i++ {
			r.half[i].free()
			r.quarter[i].free()
		}
		if r.toFinal {
			r.final.free()
		}
	}
	r.w, r.h = w, h
	gl.GenFramebuffers(1, &r.msFBO)
	gl.BindFramebuffer(gl.FRAMEBUFFER, r.msFBO)
	gl.GenRenderbuffers(1, &r.msColor)
	gl.BindRenderbuffer(gl.RENDERBUFFER, r.msColor)
	gl.RenderbufferStorageMultisample(gl.RENDERBUFFER, r.samples, r.color.internal, w, h)
	gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.RENDERBUFFER, r.msColor)
	gl.GenRenderbuffers(1, &r.msDepth)
	gl.BindRenderbuffer(gl.RENDERBUFFER, r.msDepth)
	gl.RenderbufferStorageMultisample(gl.RENDERBUFFER, r.samples, gl.DEPTH_COMPONENT24, w, h)
	gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.RENDERBUFFER, r.msDepth)
	if st := gl.CheckFramebufferStatus(gl.FRAMEBUFFER); st != gl.FRAMEBUFFER_COMPLETE {
		panic(fmt.Sprintf("scene framebuffer incomplete: 0x%x", st))
	}
	r.scene = newTarget(w, h, r.color)
	for i := 0; i < 2; i++ {
		r.half[i] = newTarget(max(w/2, 1), max(h/2, 1), r.color)
		r.quarter[i] = newTarget(max(w/4, 1), max(h/4, 1), r.color)
	}
	if r.toFinal {
		r.final = newTarget(w, h, fmtLDR)
	}
}

// --- frame ---

func (r *Renderer) setCamera(g *Game) {
	p := &g.P
	f, rt, up := p.Front(), p.Right(), p.Up()
	var eye, target V3
	upVec := V3{0, 1, 0}
	r.cockpit = false
	fov := float32(0.95)
	if g.State == stateMenu || g.State == stateName {
		eye = p.Pos.Add(rt.Mul(15)).Add(up.Mul(4.5)).Add(f.Mul(-21))
		target = p.Pos.Add(f.Mul(4)).Sub(up.Mul(2.6))
		fov = 0.8
	} else if g.Cockpit && g.State != stateGameOver {
		r.cockpit = true
		if p.BoostT > 0 {
			fov += 0.22 * float32(math.Sin(float64(p.BoostT/boostTime)*math.Pi))
		}
		eye = p.Pos.Add(f.Mul(1.5)).Add(up.Mul(1.1))
		target = eye.Add(f.Mul(100))
		// the view banks and rolls with the ship
		a := float64(p.Roll*0.6 + p.bank() - p.Roll)
		upVec = up.Mul(float32(math.Cos(a))).Sub(rt.Mul(float32(math.Sin(a))))
	} else {
		back := float32(25)
		if p.BoostT > 0 {
			k := float32(math.Sin(float64(p.BoostT/boostTime) * math.Pi))
			back += 9 * k
			fov += 0.22 * k
		}
		eye = p.Pos.Sub(f.Mul(back)).Add(up.Mul(7))
		target = p.Pos.Add(f.Mul(140)).Add(up.Mul(6))
	}
	if g.Shake > 0.01 && g.State != statePaused {
		eye = eye.Add(V3{rndS(), rndS(), rndS()}.Mul(g.Shake * 2.2))
	}
	r.fov += (fov - r.fov) * 0.15
	view := mgl.LookAtV(eye, target, upVec)
	proj := mgl.Perspective(r.fov, float32(r.w)/float32(r.h), 0.5, 14000)
	r.vp = proj.Mul4(view)
	r.camPos = eye
	r.camF = norm(target.Sub(eye))
	r.camR = norm(r.camF.Cross(upVec))
	r.camU = norm(r.camR.Cross(r.camF))
}

func (r *Renderer) drawMesh(m Mesh, model mgl.Mat4) {
	r.mesh.m4("uModel", model)
	gl.BindVertexArray(m.vao)
	gl.DrawArrays(gl.TRIANGLES, 0, m.n)
}

func (r *Renderer) meshParams(tint V3, flash, emis, spec, amb float32, rim V3, rimK float32) {
	r.mesh.v3("uTint", tint)
	r.mesh.f1("uFlash", flash)
	r.mesh.f1("uEmis", emis)
	r.mesh.f1("uSpec", spec)
	r.mesh.f1("uAmb", amb)
	r.mesh.f4("uRim", rim[0], rim[1], rim[2], rimK)
}

// billboard pushes a camera-facing quad.
func (r *Renderer) billboard(c V3, size float32, col V3, a, shape float32) {
	rx, uy := r.camR.Mul(size), r.camU.Mul(size)
	r.pquad(c.Sub(rx).Sub(uy), c.Add(rx).Sub(uy), c.Add(rx).Add(uy), c.Sub(rx).Add(uy), col, a, shape)
}

// beam pushes a quad stretched from a to b that turns to face the camera.
func (r *Renderer) beam(a, b V3, width float32, col V3, alpha float32) {
	axis := b.Sub(a)
	if axis.Len() < 1e-4 {
		return
	}
	dir := axis.Normalize()
	side := dir.Cross(r.camPos.Sub(a.Add(b).Mul(0.5)))
	if side.Len() < 1e-4 {
		return
	}
	side = side.Normalize().Mul(width)
	ext := dir.Mul(width)
	a, b = a.Sub(ext), b.Add(ext)
	r.pquad(a.Sub(side), b.Sub(side), b.Add(side), a.Add(side), col, alpha, 2)
}

func (r *Renderer) pquad(p0, p1, p2, p3 V3, col V3, a, shape float32) {
	v := func(p V3, u, w float32) {
		r.parts.data = append(r.parts.data, p[0], p[1], p[2], u, w, shape, col[0], col[1], col[2], a)
	}
	v(p0, -1, -1)
	v(p1, 1, -1)
	v(p2, 1, 1)
	v(p0, -1, -1)
	v(p2, 1, 1)
	v(p3, -1, 1)
}

func (r *Renderer) Draw(g *Game) {
	r.setCamera(g)
	p := &g.P

	gl.BindFramebuffer(gl.FRAMEBUFFER, r.msFBO)
	gl.Viewport(0, 0, r.w, r.h)
	gl.DepthMask(true)
	gl.ClearColor(0, 0, 0, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	// --- sky ---
	gl.Disable(gl.DEPTH_TEST)
	gl.DepthMask(false)
	gl.Disable(gl.BLEND)
	th := float32(math.Tan(float64(r.fov) / 2))
	r.sky.use()
	r.sky.v3("uF", r.camF)
	r.sky.v3("uR", r.camR)
	r.sky.v3("uU", r.camU)
	r.sky.f2("uTan", th*float32(r.w)/float32(r.h), th)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_CUBE_MAP, r.skyTex)
	r.sky.i1("uSky", 0)
	gl.BindVertexArray(r.triVAO)
	gl.DrawArrays(gl.TRIANGLES, 0, 3)

	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.ONE, gl.ONE)
	r.star.use()
	r.star.m4("uVP", r.vp)
	r.star.f1("uTime", g.Time)
	r.star.f1("uScale", float32(r.h)/900)
	gl.BindVertexArray(r.starVAO)
	gl.DrawArrays(gl.POINTS, 0, r.starN)
	gl.Disable(gl.BLEND)

	// --- planet (drawn at a fixed direction, far away) ---
	r.mesh.use()
	r.mesh.m4("uVP", r.vp)
	r.mesh.v3("uSun", sunDir)
	r.mesh.v3("uCam", r.camPos)
	gl.Enable(gl.CULL_FACE)
	pp := r.camPos.Add(planetDir.Mul(5200))
	r.meshParams(V3{1, 1, 1}, 0, 0, 0, 0.12, V3{0.35, 0.6, 1.0}, 1.6)
	r.drawMesh(r.mPlanet, mgl.Translate3D(pp[0], pp[1], pp[2]).
		Mul4(mgl.HomogRotate3DZ(0.35)).Mul4(mgl.HomogRotate3DY(g.Time*0.004)).Mul4(mgl.Scale3D(1500, 1500, 1500)))
	gl.Disable(gl.CULL_FACE)

	// --- opaque world ---
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthMask(true)
	r.meshParams(V3{1, 1, 1}, 0, 0, 0.08, 1, V3{0.5, 0.45, 0.6}, 0.35)
	for i := range g.Asteroids {
		a := &g.Asteroids[i]
		r.drawMesh(r.mAsteroid[a.Mesh], mgl.Translate3D(a.Pos[0], a.Pos[1], a.Pos[2]).
			Mul4(mgl.HomogRotate3D(a.Ang, a.Axis)).Mul4(mgl.Scale3D(a.Scale, a.Scale, a.Scale)))
	}
	for _, e := range g.Enemies {
		d := kinds[e.Kind]
		flash := float32(0)
		if e.Flash > 0 {
			flash = 0.5
		}
		pulse := 3.2 + 1.0*float32(math.Sin(float64(g.Time*6+e.phase)))
		r.meshParams(V3{1, 1, 1}, flash, pulse, 0.5, 1, d.col, 0.9)
		r.drawMesh(r.mEnemy[e.Kind], mgl.Translate3D(e.Pos[0], e.Pos[1], e.Pos[2]).
			Mul4(orient(e.Fwd, V3{0, 1, 0})).Mul4(mgl.Scale3D(d.scale, d.scale, d.scale)))
	}
	for i := range g.Rockets {
		rk := &g.Rockets[i]
		tint := V3{1, 1, 1}
		if rk.Hostile {
			tint = V3{1, 0.45, 0.4}
		}
		r.meshParams(tint, 0, 4, 0.5, 1, V3{1, 0.6, 0.3}, 0.6)
		r.drawMesh(r.mRocket, mgl.Translate3D(rk.Pos[0], rk.Pos[1], rk.Pos[2]).
			Mul4(orient(rk.Dir, V3{0, 1, 0})).Mul4(mgl.Scale3D(1.7, 1.7, 1.7)))
	}
	for _, pu := range g.Pickups {
		s := 3.2 + 0.4*float32(math.Sin(float64(g.Time*5)))
		r.meshParams(puCols[pu.Kind], 0, 3.5, 0, 1, V3{}, 0)
		r.drawMesh(r.mPickup, mgl.Translate3D(pu.Pos[0], pu.Pos[1], pu.Pos[2]).
			Mul4(mgl.HomogRotate3DY(g.Time*2.5)).Mul4(mgl.Scale3D(s, s, s)))
	}
	if g.State != stateGameOver && !r.cockpit {
		blink := p.Invuln > 0 && int(g.Time*18)%2 == 0
		if !blink {
			emis := float32(3.5)
			if p.BoostT > 0 {
				emis = 6
			}
			r.meshParams(V3{1, 1, 1}, 0, emis, 0.7, 1, V3{0.35, 0.7, 1.0}, 0.8)
			r.drawMesh(r.mPlayer, p.Model())
		}
	}

	// --- additive effects ---
	for _, pt := range g.Parts {
		k := pt.Life / pt.Max
		r.billboard(pt.Pos, pt.Size, pt.Col, pt.Bright*k*float32(math.Sqrt(float64(k))), pt.Shape)
	}
	pm := p.Model()
	for _, s := range g.Shots {
		dir := norm(s.Vel)
		if s.Hostile {
			r.billboard(s.Pos, 6.5, s.Col, 3.0, 0)
			r.billboard(s.Pos, 2.4, V3{1, 1, 1}, 4.0, 0)
			r.beam(s.Pos.Sub(dir.Mul(12)), s.Pos, 1.6, s.Col, 2.0)
		} else {
			// The streak grows out of the cannon that fired it. A fixed-length tail
			// would reach back through the ship and past the chase camera, so new
			// shots would look like they come out of the screen.
			muzzle := pm.Mul4x1(playerGuns[s.Gun].Vec4(1)).Vec3()
			l := mgl.Clamp(s.Pos.Sub(muzzle).Dot(dir), 0, 38)
			tail := s.Pos.Sub(dir.Mul(l))
			r.beam(tail, s.Pos, 2.6, s.Col, 2.6)
			r.beam(tail, s.Pos, 0.8, V3{1, 1, 1}, 5.0)
		}
	}
	for i := range g.Rockets {
		rk := &g.Rockets[i]
		col := V3{1, 0.65, 0.25}
		if rk.Hostile {
			col = V3{1, 0.2, 0.15}
		}
		r.billboard(rk.Pos.Sub(rk.Dir.Mul(4)), 6*(0.85+0.3*rnd()), col, 2.2, 0)
	}
	for _, pu := range g.Pickups {
		r.billboard(pu.Pos, 16, puCols[pu.Kind], 0.9, 0)
		r.billboard(pu.Pos, 9+2*float32(math.Sin(float64(g.Time*4))), puCols[pu.Kind], 1.2, 1)
	}
	for _, e := range g.Enemies {
		d := kinds[e.Kind]
		r.billboard(e.Pos.Sub(e.Fwd.Mul(d.tail*d.scale)), 4.5*d.scale, d.col, 0.9, 0)
	}
	if g.State != stateGameOver && !r.cockpit {
		m := p.Model()
		for _, e := range playerEngines {
			ep := m.Mul4x1(e.Vec4(1)).Vec3()
			size, col := float32(3.2), V3{0.25, 0.7, 1}
			if p.BoostT > 0 {
				size, col = 6, V3{1, 0.65, 0.3}
			}
			r.billboard(ep, size*(0.9+0.2*rnd()), col, 0.9, 0)
		}
	}
	// space dust streaks give a sense of speed
	const box = 400
	streak := p.Vel.Mul(0.045)
	for _, d := range g.Dust {
		var q V3
		for k := 0; k < 3; k++ {
			v := float32(math.Mod(float64(d[k]-r.camPos[k]), box))
			if v < 0 {
				v += box
			}
			q[k] = r.camPos[k] + v - box/2
		}
		if streak.Len() > 1.2 {
			r.beam(q, q.Sub(streak), 0.2, V3{0.6, 0.8, 1}, 0.5)
		} else {
			r.billboard(q, 0.35, V3{0.6, 0.8, 1}, 0.6, 0)
		}
	}
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.ONE, gl.ONE)
	gl.DepthMask(false)
	r.part.use()
	r.part.m4("uVP", r.vp)
	r.parts.flush()
	gl.DepthMask(true)
	gl.Disable(gl.BLEND)
	gl.Disable(gl.DEPTH_TEST)

	// --- resolve + bloom ---
	gl.BindFramebuffer(gl.READ_FRAMEBUFFER, r.msFBO)
	gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, r.scene.fbo)
	gl.BlitFramebuffer(0, 0, r.w, r.h, 0, 0, r.w, r.h, gl.COLOR_BUFFER_BIT, gl.NEAREST)

	gl.BindVertexArray(r.triVAO)
	gl.ActiveTexture(gl.TEXTURE0)
	down := func(src Target, dst Target, thresh float32) {
		dst.bind()
		r.bright.use()
		r.bright.i1("uTex", 0)
		r.bright.f2("uTexel", 0.5/float32(src.w), 0.5/float32(src.h))
		r.bright.f1("uThresh", thresh)
		gl.BindTexture(gl.TEXTURE_2D, src.tex)
		gl.DrawArrays(gl.TRIANGLES, 0, 3)
	}
	blur := func(t *[2]Target, passes int) {
		r.blur.use()
		r.blur.i1("uTex", 0)
		for i := 0; i < passes; i++ {
			t[1].bind()
			r.blur.f2("uDir", 1/float32(t[0].w), 0)
			gl.BindTexture(gl.TEXTURE_2D, t[0].tex)
			gl.DrawArrays(gl.TRIANGLES, 0, 3)
			t[0].bind()
			r.blur.f2("uDir", 0, 1/float32(t[0].h))
			gl.BindTexture(gl.TEXTURE_2D, t[1].tex)
			gl.DrawArrays(gl.TRIANGLES, 0, 3)
		}
	}
	down(r.scene, r.half[0], 1.0)
	blur(&r.half, 1)
	down(r.half[0], r.quarter[0], 0)
	blur(&r.quarter, 2)

	// --- composite ---
	if r.toFinal {
		r.final.bind()
	} else {
		gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
		gl.Viewport(0, 0, r.outW, r.outH) // scales a smaller scene up; the HUD then draws at full resolution
	}
	r.comp.use()
	r.comp.i1("uScene", 0)
	r.comp.i1("uB1", 1)
	r.comp.i1("uB2", 2)
	gl.ActiveTexture(gl.TEXTURE2)
	gl.BindTexture(gl.TEXTURE_2D, r.quarter[0].tex)
	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, r.half[0].tex)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, r.scene.tex)
	ab := 0.003 + g.Shake*0.04
	if p.BoostT > 0 {
		ab += 0.05
	}
	r.comp.f1("uAb", ab)
	r.comp.f1("uTime", g.Time)
	r.comp.f1("uHit", g.HitFx*0.9)
	r.comp.v3("uHitCol", V3{1.0, 0.12, 0.08})
	fade := float32(1)
	if g.State == statePaused {
		fade = 0.45
	}
	r.comp.f1("uFade", fade)
	gl.DrawArrays(gl.TRIANGLES, 0, 3)
}

// project maps a world point to pixel coordinates; ok is false behind the camera.
func (r *Renderer) project(p V3) (x, y float32, ok bool) {
	c := r.vp.Mul4x1(p.Vec4(1))
	if c[3] <= 0.01 {
		return 0, 0, false
	}
	return (c[0]/c[3]*0.5 + 0.5) * float32(r.w), (0.5 - c[1]/c[3]*0.5) * float32(r.h), true
}
