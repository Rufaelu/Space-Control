package main

import (
	"math"
	"testing"

	mgl "github.com/go-gl/mathgl/mgl32"
)

func near(a, b float32) bool { return math.Abs(float64(a-b)) < 0.03 }

func newFlight() *Game {
	g := &Game{}
	g.Reset()
	return g
}

// fly runs the ship for secs, calling look(dt) every frame for mouse input.
func fly(g *Game, secs float32, look func(dt float32)) {
	const dt = 1.0 / 60
	for t := float32(0); t < secs; t += dt {
		if look != nil {
			look(dt)
		}
		g.updatePlayer(dt, Input{})
	}
}

func TestHeadings(t *testing.T) {
	p := Player{Rot: yawQuat(-90)}
	if f := p.Front(); !near(f.X(), 0) || !near(f.Y(), 0) || !near(f.Z(), -1) {
		t.Fatalf("yaw -90 faces %v, want -Z", f)
	}
	if r := p.Right(); !near(r.X(), 1) {
		t.Fatalf("right wing %v, want +X", r)
	}
	p.Rot = yawQuat(0)
	if f := p.Front(); !near(f.X(), 1) {
		t.Fatalf("yaw 0 faces %v, want +X", f)
	}
}

func TestMouseDirections(t *testing.T) {
	g := newFlight()
	g.Look(10, 0)
	if f := g.P.Front(); f.X() <= 0.1 {
		t.Fatalf("mouse right turned to %v", f)
	}
	g = newFlight()
	g.Look(0, -10) // mouse up
	if f := g.P.Front(); f.Y() <= 0.1 {
		t.Fatalf("mouse up pitched to %v", f)
	}
}

// Holding the mouse up keeps pitching: over the top, upside down heading
// back, and round to where it started.
func TestFullLoop(t *testing.T) {
	g := newFlight()
	const rate = 120.0 // degrees per second
	pull := func(dt float32) { g.Look(0, -rate*dt) }
	fly(g, 1.5, pull) // 180 degrees: over the top
	f, u := g.P.Front(), g.P.Up()
	if f.Z() < 0.95 || u.Y() > -0.95 {
		t.Fatalf("half loop: front %v up %v, want heading back (+Z) and inverted", f, u)
	}
	fly(g, 1.5, pull)
	f, u = g.P.Front(), g.P.Up()
	if f.Z() > -0.95 || u.Y() < 0.95 {
		t.Fatalf("full loop: front %v up %v, want back to -Z upright", f, u)
	}
}

// Let go at the top of a loop and the ship rolls upright still heading
// back the way it came: an Immelmann, onto the tail of whoever chased you.
func TestImmelmann(t *testing.T) {
	g := newFlight()
	fly(g, 1.5, func(dt float32) { g.Look(0, -120*dt) })
	fly(g, 2, nil)
	f, u := g.P.Front(), g.P.Up()
	if f.Z() < 0.95 || u.Y() < 0.95 {
		t.Fatalf("after letting go: front %v up %v, want heading +Z upright", f, u)
	}
}

func TestLevelsAfterRoll(t *testing.T) {
	for _, a := range []float32{0.5, -0.5, 2.5} {
		g := newFlight()
		g.P.Rot = g.P.Rot.Mul(qRoll(a))
		if r := g.P.rollAngle(); !near(r, a) {
			t.Fatalf("rolled %v right, rollAngle says %v", a, r)
		}
		fly(g, 3, nil)
		if r := g.P.rollAngle(); math.Abs(float64(r)) > 0.02 {
			t.Fatalf("rolled %v: still %v off level after 3 s", a, r)
		}
	}
}

// qRoll rolls the ship by a radians about its nose, right wing down.
func qRoll(a float32) mgl.Quat { return mgl.QuatRotate(a, V3{0, 0, -1}) }
