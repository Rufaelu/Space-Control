package main

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/png"
	"math"
	"math/rand"
	"os"
	"runtime"
	"time"

	"github.com/go-gl/gl/v4.1-core/gl"
	mgl "github.com/go-gl/mathgl/mgl32"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	laserSpeed  = 600.0
	enemySpeed  = 26.0
	fireRate    = 0.12
	maxEnemies  = 100
	maxHeat     = 1.0
	heatPerShot = 0.15
	coolRate    = 0.4
)

type GameState int

const (
	stateRunning GameState = iota
	statePaused
	stateGameOver
	stateWin
)

type Camera struct {
	Pos              mgl.Vec3
	Yaw, Pitch, Sens float32
}

func (c *Camera) Front() mgl.Vec3 {
	y, p := mgl.DegToRad(c.Yaw), mgl.DegToRad(c.Pitch)
	return mgl.Vec3{
		float32(math.Cos(float64(y)) * math.Cos(float64(p))),
		float32(math.Sin(float64(p))),
		float32(math.Sin(float64(y)) * math.Cos(float64(p))),
	}.Normalize()
}
func (c *Camera) Right() mgl.Vec3 { return c.Front().Cross(mgl.Vec3{0, 1, 0}).Normalize() }

// --- SHADERS ---
const vertexShaderSrc = `#version 330 core
layout (location = 0) in vec3 aPos;
layout (location = 1) in vec2 aTexCoord;
uniform mat4 uMVP;
out vec2 vTexCoord;
void main() {
    vTexCoord = aTexCoord;
    gl_Position = uMVP * vec4(aPos, 1.0);
}` + "\x00"

const fragmentShaderSrc = `#version 330 core
out vec4 FragColor;
in vec2 vTexCoord;
uniform sampler2D uTexture;
uniform vec3 uColor;
uniform int uRenderMode;
void main() {
    if (uRenderMode == 1) {
        FragColor = texture(uTexture, vTexCoord);
    } else if (uRenderMode == 2) {
        FragColor = vec4(1.0, 1.0, 1.0, 1.0);
    } else {
        FragColor = vec4(uColor, 1.0);
    }
}` + "\x00"

type Enemy struct {
	Pos   mgl.Vec3
	Alive bool
}
type Laser struct {
	Pos, Dir mgl.Vec3
	Rot      mgl.Mat4
	Life     float32
}

func main() {
	runtime.LockOSThread()
	sdl.Init(sdl.INIT_VIDEO)

	window, err := sdl.CreateWindow("SPACE CONTROL", sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_CENTERED, 1280, 720, sdl.WINDOW_OPENGL|sdl.WINDOW_MAXIMIZED|sdl.WINDOW_RESIZABLE)
	if err != nil {
		panic(err)
	}

	glctx, _ := window.GLCreateContext()
	window.GLMakeCurrent(glctx)
	gl.Init()
	gl.Enable(gl.DEPTH_TEST)

	prog := createProgram(vertexShaderSrc, fragmentShaderSrc)
	vao, _ := setupBuffers()
	enemyTex, _ := loadTexture("enemy.png")

	uMVP := gl.GetUniformLocation(prog, gl.Str("uMVP\x00"))
	uColor := gl.GetUniformLocation(prog, gl.Str("uColor\x00"))
	uMode := gl.GetUniformLocation(prog, gl.Str("uRenderMode\x00"))

	cam := Camera{Pos: mgl.Vec3{0, 0, 0}, Yaw: -90, Sens: 0.15}
	sdl.SetRelativeMouseMode(true)

	enemies, lasers, stars := []*Enemy{}, []*Laser{}, []mgl.Vec3{}
	for i := 0; i < 1500; i++ {
		stars = append(stars, mgl.Vec3{rand.Float32()*8000 - 4000, rand.Float32()*8000 - 4000, rand.Float32()*8000 - 4000})
	}

	score, spawnedCount, shields, currentState := 0, 0, 3, stateRunning
	weaponHeat, overheated := float32(0.0), false
	last := time.Now()
	var lastShot time.Time
	var lastTitleUpdate time.Time
	var hitFlashTimer float32

	for {
		dt := float32(time.Since(last).Seconds())
		if dt > 0.1 {
			dt = 0.1
		}
		last = time.Now()
		winW, winH := window.GetSize()

		// --- HUD LOGIC ---
		if time.Since(lastTitleUpdate).Seconds() > 0.1 {
			shieldIcons := ""
			for i := 0; i < shields; i++ {
				shieldIcons += "■"
			}
			heatPct := int(weaponHeat * 100)

			status := fmt.Sprintf("HEAT: %d%%", heatPct)
			if overheated {
				status = "!!! OVERHEATED !!!"
			}
			if currentState == statePaused {
				status = "|| PAUSED ||"
			}
			if currentState == stateGameOver {
				status = "GAME OVER (R to restart)"
			}
			if currentState == stateWin {
				status = "YOU WIN! (R to restart)"
			}

			title := fmt.Sprintf("SPACE CONTROL | %s | Score: %d | Shields: %s", status, score, shieldIcons)
			window.SetTitle(title)
			lastTitleUpdate = time.Now()
		}

		// --- EVENT LOOP ---
		for ev := sdl.PollEvent(); ev != nil; ev = sdl.PollEvent() {
			switch e := ev.(type) {
			case *sdl.QuitEvent:
				return
			case *sdl.MouseMotionEvent:
				if currentState == stateRunning {
					cam.Yaw += float32(e.XRel) * cam.Sens
					cam.Pitch = mgl.Clamp(cam.Pitch-float32(e.YRel)*cam.Sens, -89, 89)
				}
			case *sdl.KeyboardEvent:
				if e.Type == sdl.KEYDOWN {
					if e.Keysym.Sym == sdl.K_ESCAPE {
						return
					}
					if e.Keysym.Sym == sdl.K_p {
						if currentState == stateRunning {
							currentState = statePaused
							sdl.SetRelativeMouseMode(false)
						} else if currentState == statePaused {
							currentState = stateRunning
							sdl.SetRelativeMouseMode(true)
						}
					}
					if e.Keysym.Sym == sdl.K_r && (currentState == stateGameOver || currentState == stateWin) {
						enemies, lasers, score, spawnedCount, shields, currentState = []*Enemy{}, []*Laser{}, 0, 0, 3, stateRunning
						weaponHeat, overheated = 0, false
						cam.Pos = mgl.Vec3{0, 0, 0}
						sdl.SetRelativeMouseMode(true)
					}
				}
			}
		}

		// --- WORLD UPDATE ---
		if currentState == stateRunning {
			if weaponHeat > 0 {
				weaponHeat -= coolRate * dt
			} else {
				weaponHeat = 0
				overheated = false
			}

			keys := sdl.GetKeyboardState()
			_, _, mState := sdl.GetMouseState()
			f, r := cam.Front(), cam.Right()
			spd := float32(130.0 * dt)

			if keys[sdl.SCANCODE_W] != 0 {
				cam.Pos = cam.Pos.Add(f.Mul(spd))
			}
			if keys[sdl.SCANCODE_S] != 0 {
				cam.Pos = cam.Pos.Sub(f.Mul(spd))
			}
			if keys[sdl.SCANCODE_A] != 0 {
				cam.Pos = cam.Pos.Sub(r.Mul(spd))
			}
			if keys[sdl.SCANCODE_D] != 0 {
				cam.Pos = cam.Pos.Add(r.Mul(spd))
			}

			if (keys[sdl.SCANCODE_SPACE] != 0 || (mState&sdl.Button(sdl.BUTTON_LEFT) != 0)) &&
				time.Since(lastShot).Seconds() > fireRate && !overheated {
				weaponHeat += heatPerShot
				if weaponHeat >= maxHeat {
					overheated = true
				}
				rot := mgl.QuatBetweenVectors(mgl.Vec3{0, 0, -1}, f).Mat4()
				spawnPos := cam.Pos.Add(f.Mul(35.0))
				lasers = append(lasers, &Laser{Pos: spawnPos, Dir: f, Rot: rot, Life: 2.0})
				lastShot = time.Now()
			}

			if spawnedCount < maxEnemies && len(enemies) < 20 && rand.Float32() < 0.02 {
				offset := f.Mul(650).Add(mgl.Vec3{rand.Float32()*500 - 250, rand.Float32()*500 - 250, rand.Float32()*200 - 100})
				enemies = append(enemies, &Enemy{Pos: cam.Pos.Add(offset), Alive: true})
				spawnedCount++
			}

			if spawnedCount >= maxEnemies && len(enemies) == 0 {
				currentState = stateWin
				sdl.SetRelativeMouseMode(false)
			}

			nl := lasers[:0]
			for _, l := range lasers {
				l.Pos = l.Pos.Add(l.Dir.Mul(laserSpeed * dt))
				l.Life -= dt
				hit := false
				for _, en := range enemies {
					if en.Alive && en.Pos.Sub(l.Pos).Len() < 12.0 {
						en.Alive, hit, score = false, true, score+1
					}
				}
				if l.Life > 0 && !hit {
					nl = append(nl, l)
				}
			}
			lasers = nl

			ne := enemies[:0]
			for _, en := range enemies {
				en.Pos = en.Pos.Add(cam.Pos.Sub(en.Pos).Normalize().Mul(enemySpeed * dt))
				if en.Pos.Sub(cam.Pos).Len() < 8.0 {
					shields--
					en.Alive = false
					hitFlashTimer = 0.12
					if shields <= 0 {
						currentState = stateGameOver
						sdl.SetRelativeMouseMode(false)
					}
				}
				if en.Alive {
					ne = append(ne, en)
				}
			}
			enemies = ne
		}

		// --- RENDER ---
		gl.Viewport(0, 0, int32(winW), int32(winH))
		if hitFlashTimer > 0 {
			gl.ClearColor(0, 0.2, 0.5, 1)
			hitFlashTimer -= dt
		} else if currentState == statePaused {
			gl.ClearColor(0.02, 0.02, 0.1, 1)
		} else {
			gl.ClearColor(0.0, 0.0, 0.03, 1)
		}
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		gl.UseProgram(prog)

		view := mgl.LookAtV(cam.Pos, cam.Pos.Add(cam.Front()), mgl.Vec3{0, 1, 0})
		vp := mgl.Perspective(0.8, float32(winW)/float32(winH), 0.1, 10000.0).Mul4(view)
		gl.BindVertexArray(vao)

		// Stars
		gl.Uniform1i(uMode, 2)
		for _, s := range stars {
			m := mgl.Translate3D(s.X()+cam.Pos.X()*0.995, s.Y()+cam.Pos.Y()*0.995, s.Z()+cam.Pos.Z()*0.995)
			mvp := vp.Mul4(m)
			gl.UniformMatrix4fv(uMVP, 1, false, &mvp[0])
			gl.DrawElements(gl.TRIANGLES, 36, gl.UNSIGNED_INT, nil)
		}

		// Enemies
		gl.Uniform1i(uMode, 1)
		gl.BindTexture(gl.TEXTURE_2D, enemyTex)
		for _, en := range enemies {
			m := mgl.Translate3D(en.Pos.X(), en.Pos.Y(), en.Pos.Z()).Mul4(mgl.Scale3D(18, 18, 18))
			mvp := vp.Mul4(m)
			gl.UniformMatrix4fv(uMVP, 1, false, &mvp[0])
			gl.DrawElements(gl.TRIANGLES, 36, gl.UNSIGNED_INT, nil)
		}

		// Lasers
		gl.Uniform1i(uMode, 0)
		laserCol := mgl.Vec3{0.0, 0.8, 1.0}
		if overheated {
			laserCol = mgl.Vec3{1.0, 0.2, 0.2}
		}
		gl.Uniform3f(uColor, laserCol.X(), laserCol.Y(), laserCol.Z())
		for _, l := range lasers {
			m := mgl.Translate3D(l.Pos.X(), l.Pos.Y(), l.Pos.Z()).Mul4(l.Rot).Mul4(mgl.Scale3D(0.4, 0.4, 60.0))
			mvp := vp.Mul4(m)
			gl.UniformMatrix4fv(uMVP, 1, false, &mvp[0])
			gl.DrawElements(gl.TRIANGLES, 36, gl.UNSIGNED_INT, nil)
		}
		window.GLSwap()
	}
}

// --- UTILS (REMAIN THE SAME) ---

func loadTexture(file string) (uint32, error) {
	f, err := os.Open(file)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return 0, err
	}
	rgba := image.NewRGBA(img.Bounds())
	draw.Draw(rgba, rgba.Bounds(), img, image.Pt(0, 0), draw.Src)
	var t uint32
	gl.GenTextures(1, &t)
	gl.BindTexture(gl.TEXTURE_2D, t)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(rgba.Rect.Size().X), int32(rgba.Rect.Size().Y), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(rgba.Pix))
	return t, nil
}

func setupBuffers() (uint32, int32) {
	vertices := []float32{
		-0.5, -0.5, 0.5, 0, 0, 0.5, -0.5, 0.5, 1, 0, 0.5, 0.5, 0.5, 1, 1, -0.5, 0.5, 0.5, 0, 1,
		-0.5, -0.5, -0.5, 0, 0, 0.5, -0.5, -0.5, 1, 0, 0.5, 0.5, -0.5, 1, 1, -0.5, 0.5, -0.5, 0, 1,
		-0.5, 0.5, 0.5, 1, 0, -0.5, 0.5, -0.5, 1, 1, -0.5, -0.5, -0.5, 0, 1, -0.5, -0.5, 0.5, 0, 0,
		0.5, 0.5, 0.5, 1, 0, 0.5, 0.5, -0.5, 1, 1, 0.5, -0.5, -0.5, 0, 1, 0.5, -0.5, 0.5, 0, 0,
		-0.5, 0.5, 0.5, 0, 0, 0.5, 0.5, 0.5, 1, 0, 0.5, 0.5, -0.5, 1, 1, -0.5, 0.5, -0.5, 0, 1,
		-0.5, -0.5, 0.5, 0, 0, 0.5, -0.5, 0.5, 1, 0, 0.5, -0.5, -0.5, 1, 1, -0.5, -0.5, -0.5, 0, 1,
	}
	indices := []uint32{0, 1, 2, 2, 3, 0, 4, 5, 6, 6, 7, 4, 8, 9, 10, 10, 11, 8, 12, 13, 14, 14, 15, 12, 16, 17, 18, 18, 19, 16, 20, 21, 22, 22, 23, 20}
	var vao, vbo, ebo uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)
	gl.GenBuffers(1, &ebo)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, ebo)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(indices)*4, gl.Ptr(indices), gl.STATIC_DRAW)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 5*4, nil)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, 5*4, gl.PtrOffset(3*4))
	gl.EnableVertexAttribArray(1)
	return vao, 36
}

func createProgram(v, f string) uint32 {
	vsh, fsh := compileShader(v, gl.VERTEX_SHADER), compileShader(f, gl.FRAGMENT_SHADER)
	p := gl.CreateProgram()
	gl.AttachShader(p, vsh)
	gl.AttachShader(p, fsh)
	gl.LinkProgram(p)
	return p
}

func compileShader(src string, typ uint32) uint32 {
	s := gl.CreateShader(typ)
	cstr, free := gl.Strs(src)
	gl.ShaderSource(s, 1, cstr, nil)
	free()
	gl.CompileShader(s)
	return s
}
