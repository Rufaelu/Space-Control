# 🚀 Space Control

A fast-paced, skill-based **3D space shooter** built in **Go** using **OpenGL** and **SDL2**.  
Navigate a vast starfield, manage your weapon heat, and defend your ship against waves of aggressive drones.

---

## 🎮 Gameplay Video

```html
<video src="GamePlay.mp4" controls width="800"></video>
```

---

## 🛠 Features

- ✅ **Custom 3D Engine** built from scratch using Core Profile OpenGL 4.1
- 🔥 **Weapon Heat Mechanic** — rapid firing causes overheating and weapon lockout
- 🎯 **Precision Collision System** — tight hitboxes reward accurate aim and narrow dodges
- 🛡 **Shield System** — three-stage hull protection with damage flash feedback
- 🎮 **Responsive 6-DOF Controls** — mouse look + keyboard strafing
- ✨ **Procedural Starfield** — over 1,500 stars rendered for deep space immersion

---

## 🕹 Controls

| Key | Action |
|-----|--------|
| **W / A / S / D** | Move Ship (Forward, Left, Back, Right) |
| **Mouse Move** | Aim / Look Around |
| **Left Click / Space** | Fire Lasers |
| **P** | Toggle Pause (Unlocks mouse) |
| **R** | Restart (After Game Over / Victory) |
| **Esc** | Quit Game |

---

## 🚀 Installation & Running

### 1️⃣ Prerequisites

Make sure you have:

- [Go (Golang)](https://golang.org/dl/)
- A C compiler (GCC / Clang) for CGO
- SDL2 development libraries installed on your system

#### On Linux (Debian/Kali/Ubuntu)

```bash
sudo apt install libsdl2-dev
```

---

### 2️⃣ Install Dependencies

```bash
go get github.com/go-gl/gl/v4.1-core/gl
go get github.com/go-gl/mathgl/mgl32
go get github.com/veandco/go-sdl2/sdl
```

---

### 3️⃣ Setup Assets

Place this file in the project root:

```
enemy.png
```

This texture is required for enemy drones. The game will not start without it.

---

### 4️⃣ Run the Game

```bash
go run main.go
```

---

## 📦 Building a Portable Executable

### 🪟 Windows

```bash
go build -ldflags "-H windowsgui" -o SpaceControl.exe main.go
```

### 🐧 Linux / 🍎 macOS

```bash
go build -o SpaceControl main.go
chmod +x SpaceControl
```

> ⚠️ When sharing the executable, **include `enemy.png` in the same folder**.

---

## 🏗 Built With

- **Language:** Go
- **Graphics:** OpenGL 4.1 (Core Profile)
- **Windowing & Input:** SDL2
- **Math Library:** MathGL

---

## 💡 Project Goal

This project was built to explore:

- Low-level graphics programming in Go
- OpenGL rendering pipeline
- Game loop architecture
- Input handling with SDL2
- Real-time collision detection and gameplay mechanics

---

## 📜 License

MIT License — feel free to use, modify, and learn from it.
