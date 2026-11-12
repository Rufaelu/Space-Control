# 🚀 Space Control

A fast-paced, skill-based **3D space shooter** built in **Go** using **OpenGL** and **SDL2**.  
Navigate a vast starfield, manage your weapon heat, and defend your ship against waves of aggressive drones.

---

## 🎮 Gameplay Video

[![Watch Gameplay](GamePlay.gif)](GamePlay.mp4)


---

## 🛠 Features

- ✅ **Custom 3D Engine** built from scratch on Core Profile OpenGL 3.3
- 🌌 **HDR Rendering** — bloom, tone mapping, 4x MSAA, procedural nebula, twinkling stars, a sun and a gas giant
- 🛸 **Ships built in code** — your fighter plus four enemy classes, no texture files needed
- 🎥 **Chase Camera** — your ship banks into turns; the view pulls back when you boost
- 🌊 **Endless Waves** — drones that shoot and ram, fighters that strafe and shoot, armoured bruisers, and a dreadnought boss every 5th wave
- 🔥 **Weapon Heat Mechanic** — rapid firing causes overheating and weapon lockout
- 🛡 **Shield System** — five shield cells, brief invulnerability after a hit
- ⚡ **Power-ups** — shield repair, coolant flush, rapid fire
- 🚀 **Boost** — a burst of speed on a cooldown
- 🎚 **Three levels** — Easy, Normal and Hard, picked on the title screen
- 🎯 **Homing rockets** — lock on and fire; on Hard, enemy ships launch rockets at you too (shoot them down or flip through them)
- 🙌 **Knows your name** — asks once, then congratulates you by name after every wave
- 🔄 **Flip manoeuvre** — a sideways barrel roll, left or right, that lets shots pass through
- 👁 **Two views** — chase camera or first-person cockpit
- 🏆 **Combo multiplier and saved high score**
- 🔊 **Synthesized stereo sound** — every ship class has its own engine note, placed left/right and pitched by how fast it is closing; plus lasers, explosions and alerts
- 🎵 **Original soundtrack** — a driving synth loop generated in code that picks up in combat and boss fights
- 🎧 **Your own music** — drop MP3, OGG or WAV songs in the `music` folder and the game plays them instead
- ⏸ **Pause menu** — resume, restart, volume, quit to title, quit game
- 📡 **On-screen HUD** — radar, heat gauge, target brackets, off-screen threat arrows
- ☄️ **Asteroid field** — rocks block shots and damage your ship; shoot them and they shatter into smaller pieces

---

## 🕹 Controls

| Key | Action |
|-----|--------|
| **W / A / S / D** | Thrust (Forward, Left, Back, Right) |
| **Mouse Move** | Aim / Steer |
| **Space / Left Click** | Fire Lasers |
| **Shift** | Boost |
| **Alt / Right Click** | Fire a homing rocket |
| **Q** | Flip left — a sideways barrel roll that dodges shots |
| **E** | Flip right |
| **V** | Switch between chase and cockpit view |
| **Esc / P** | Pause menu (resume, restart, volume, engine loudness, music, quit) |
| **F / F11** | Toggle Fullscreen |
| **M** | Music on/off |
| **N** | Next song (when playing your own music) |
| **A / D** (title screen) | Change level |
| **F2** (title screen) | Change your name |
| **R** | Restart (After your ship is destroyed) |

To quit, click **QUIT GAME** on the title screen, choose **QUIT GAME** in the pause menu,
or close the window. Esc never closes the game.

---

## 🎧 Your Own Music

The game plays every MP3, OGG and WAV file in its `music` folder, in random order, instead
of the built-in soundtrack. A song added while the game is running plays next; with the
folder empty, the built-in soundtrack comes back.

| Installed on | Music folder |
|--------------|--------------|
| Windows | `%LOCALAPPDATA%\Programs\SpaceControl\music` (Start menu → Space Control → Space Control Music Folder) |
| Linux | `~/.local/share/space-control/music` (applications menu → right-click Space Control → Open Music Folder) |
| Running from source | `music/` in the project folder |

Uninstalling keeps the songs you added.

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

### 3️⃣ Run the Game

```bash
go run .
```

---

## 💿 Installers

Both installers ask for your name, which the game uses to congratulate you.

### 📀 Installers for another PC (recommended)

With [Docker](https://docs.docker.com/get-docker/) running, on Linux:

```bash
sh installer/build-installers.sh            # both; or add: linux | windows
```

This builds, in containers, installers that carry everything the game needs, so nothing
has to be installed on the other PC first:

- `dist/SpaceControl-windows-x86_64-setup.exe` — Windows 10/11, 64-bit
- `dist/SpaceControl-linux-x86_64-installer.run` — 64-bit desktop Linux from 2022 on:
  Ubuntu 22.04+, Debian 12+, Fedora 36+, Mint 21+, Arch (built against glibc 2.35, with SDL2 inside)
- `dist/HOW-TO-INSTALL.txt` — install steps for the person at the other PC

Copy the `dist` folder to the other PC. The first build takes a few minutes while the
toolchains download; later builds reuse them.

### 🐧 Linux (using this machine's SDL2)

```bash
sh installer/build-linux.sh
sh dist/SpaceControl-linux-x86_64-installer.run
```

Installs for the current user (no sudo) into `~/.local/share/space-control`, adds a
`space-control` command and an applications-menu entry. Built this way the installed game
uses the system's SDL2; `installer/build-installers.sh` makes one that carries its own.
Uninstall with `~/.local/share/space-control/uninstall.sh`.

### 🪟 Windows (on a Windows machine)

In an MSYS2 MINGW64 shell with `mingw-w64-x86_64-gcc`, `mingw-w64-x86_64-go`,
`mingw-w64-x86_64-SDL2` and `mingw-w64-x86_64-nsis` installed:

```bash
sh installer/build-windows.sh
```

This produces `dist/SpaceControl-windows-x86_64-setup.exe`, a per-user setup wizard with
Start Menu and Desktop shortcuts and an uninstaller.

### ☁️ Without a Windows machine

The GitHub workflow in `.github/workflows/installers.yml` builds both installers.
It runs on every push (or from the repository's Actions tab); download the installers
from the finished run's Artifacts section.

---

## 📦 Building a Portable Executable

### 🪟 Windows

```bash
go build -ldflags "-H windowsgui" -o SpaceControl.exe .
```

### 🐧 Linux / 🍎 macOS

```bash
go build -o SpaceControl .
chmod +x SpaceControl
```

> The executable is self-contained: all ships and scenery are generated in code.

---

## 🏗 Built With

- **Language:** Go
- **Graphics:** OpenGL 3.3 (Core Profile)
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
