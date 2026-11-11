package main

import "math"

// Every ship is built in code: nose along -Z, up +Y, right +X.

// Hard points on the player model, in model space.
var (
	playerGuns    = [2]V3{{-7.5, -0.5, 0.0}, {7.5, -0.5, 0.0}}
	playerEngines = [2]V3{{-1.75, -0.15, 6.3}, {1.75, -0.15, 6.3}}
)

func buildPlayer() []float32 {
	b := &MB{}
	hull := v3(0.78, 0.81, 0.90)
	dark := v3(0.10, 0.11, 0.15)
	accent := v3(0.05, 0.55, 1.0)
	up, right := v3(0, 1, 0), v3(1, 0, 0)

	b.set(hull, 0)
	b.loft([]sec{
		{z: -9.5, w: 0.12, h: 0.10, y: -0.1},
		{z: -5.5, w: 0.80, h: 0.55},
		{z: -1.5, w: 1.35, h: 0.95, y: 0.1},
		{z: 2.5, w: 1.50, h: 1.00, y: 0.1},
		{z: 5.2, w: 1.05, h: 0.70, y: 0.1},
	}, 6)
	// canopy
	b.set(v3(0.10, 0.55, 1.0), 0.30)
	b.loft([]sec{
		{z: -4.8, w: 0.22, h: 0.12, y: 0.50},
		{z: -3.0, w: 0.62, h: 0.48, y: 0.85},
		{z: -0.8, w: 0.58, h: 0.42, y: 0.98},
		{z: 0.9, w: 0.20, h: 0.12, y: 0.95},
	}, 6)
	// dorsal accent strip
	b.set(accent, 1)
	b.box(v3(0, 1.12, 2.6), v3(0.12, 0.05, 1.6))

	for _, sx := range []float32{-1, 1} {
		b.set(hull, 0)
		b.plate(v3(sx*1.2, 0, -1.2), v3(sx*1.2, 0, 4.2), v3(sx*7.5, -0.5, 4.8), v3(sx*7.5, -0.5, 2.4), up, 0.55, 0.16)
		// wing accent
		b.set(accent, 1)
		b.plate(v3(sx*2.4, 0.22, 3.4), v3(sx*2.4, 0.22, 3.9), v3(sx*7.0, -0.28, 4.5), v3(sx*7.0, -0.28, 4.2), up, 0.08, 0.08)
		// wingtip cannon
		b.set(dark, 0)
		b.loft([]sec{
			{z: -0.4, w: 0.22, h: 0.22, x: sx * 7.5, y: -0.5},
			{z: 1.2, w: 0.48, h: 0.48, x: sx * 7.5, y: -0.5},
			{z: 5.0, w: 0.48, h: 0.48, x: sx * 7.5, y: -0.5},
			{z: 5.5, w: 0.25, h: 0.25, x: sx * 7.5, y: -0.5},
		}, 6)
		b.set(accent, 1)
		b.box(v3(sx*7.5, -0.5, -0.5), v3(0.14, 0.14, 0.22))
		// tail fin
		b.set(hull, 0)
		b.plate(v3(sx*0.9, 0.7, 2.0), v3(sx*0.9, 0.7, 5.0), v3(sx*2.0, 3.3, 5.8), v3(sx*2.0, 3.3, 4.5), right, 0.30, 0.10)
		// engine
		b.set(dark, 0)
		b.loft([]sec{
			{z: 1.8, w: 0.70, h: 0.70, x: sx * 1.75, y: -0.15},
			{z: 5.9, w: 0.92, h: 0.92, x: sx * 1.75, y: -0.15},
		}, 8)
		b.set(v3(0.25, 0.75, 1.0), 1)
		b.loft([]sec{
			{z: 5.9, w: 0.74, h: 0.74, x: sx * 1.75, y: -0.15},
			{z: 6.2, w: 0.55, h: 0.55, x: sx * 1.75, y: -0.15},
		}, 8)
	}
	return b.v
}

// Drone: a fast three-bladed dart with a single glowing eye.
func buildDrone() []float32 {
	b := &MB{}
	glow := v3(1.0, 0.28, 0.12)
	b.set(v3(0.34, 0.06, 0.07), 0)
	b.octa(v3(0, 0, 0), v3(2.0, 1.6, 4.2))
	b.set(glow, 1)
	b.octa(v3(0, 0, -3.3), v3(0.85, 0.85, 1.1))
	b.octa(v3(0, 0, 3.7), v3(0.7, 0.7, 0.9))
	for i := 0; i < 3; i++ {
		a := float64(i)*2*math.Pi/3 + math.Pi/2
		d := v3(float32(math.Cos(a)), float32(math.Sin(a)), 0)
		t := v3(-d[1], d[0], 0)
		z := func(r, zz float32) V3 { return d.Mul(r).Add(v3(0, 0, zz)) }
		b.set(v3(0.13, 0.13, 0.17), 0)
		b.plate(z(1.0, -1.8), z(1.0, 2.2), z(6.2, 5.0), z(5.6, 2.6), t, 0.55, 0.12)
		b.set(glow, 1)
		b.octa(z(6.0, 4.4), v3(0.35, 0.35, 0.7))
	}
	return b.v
}

// Fighter: forward-swept interceptor that keeps its distance and shoots.
func buildFighter() []float32 {
	b := &MB{}
	glow := v3(1.0, 0.18, 0.9)
	hull := v3(0.17, 0.14, 0.28)
	up := v3(0, 1, 0)
	b.set(hull, 0)
	b.loft([]sec{
		{z: -7.5, w: 0.10, h: 0.08},
		{z: -3.0, w: 0.95, h: 0.60},
		{z: 2.0, w: 1.25, h: 0.80},
		{z: 5.0, w: 0.75, h: 0.50},
	}, 6)
	b.set(glow, 1)
	b.octa(v3(0, 0.62, -2.4), v3(0.42, 0.28, 1.3))
	b.octa(v3(0, 0, 5.1), v3(0.6, 0.42, 0.5))
	for _, sx := range []float32{-1, 1} {
		b.set(hull, 0)
		b.plate(v3(sx*1.0, 0, 0.4), v3(sx*1.0, 0, 4.6), v3(sx*6.6, 0.9, -1.0), v3(sx*6.6, 0.9, -3.2), up, 0.5, 0.14)
		b.plate(v3(sx*0.8, -0.2, 3.0), v3(sx*0.8, -0.2, 5.0), v3(sx*2.6, -2.2, 5.8), v3(sx*2.6, -2.2, 4.6), v3(1, 0, 0), 0.25, 0.08)
		b.set(glow, 1)
		b.box(v3(sx*6.6, 0.9, -3.3), v3(0.16, 0.16, 0.9))
		b.plate(v3(sx*1.6, 0.36, 2.6), v3(sx*1.6, 0.36, 3.0), v3(sx*6.2, 1.0, -1.4), v3(sx*6.2, 1.0, -1.8), up, 0.06, 0.06)
	}
	return b.v
}

// Bruiser: an armoured gunship with twin weapon pods.
func buildBruiser() []float32 {
	b := &MB{}
	glow := v3(1.0, 0.45, 0.06)
	hull := v3(0.24, 0.22, 0.20)
	plate := v3(0.36, 0.34, 0.33)
	b.set(hull, 0)
	b.loft([]sec{
		{z: -6.5, w: 1.8, h: 1.3},
		{z: -4.0, w: 4.0, h: 3.0},
		{z: 3.0, w: 4.5, h: 3.4},
		{z: 6.5, w: 3.0, h: 2.2},
	}, 6)
	b.set(plate, 0)
	b.box(v3(0, 3.1, -0.2), v3(2.3, 0.45, 3.4))
	b.box(v3(0, -3.1, -0.2), v3(2.3, 0.45, 3.4))
	b.set(glow, 1)
	b.box(v3(0, 0, -6.55), v3(1.1, 0.45, 0.12))
	b.box(v3(0, 3.6, -0.2), v3(0.25, 0.08, 2.6))
	for _, sx := range []float32{-1, 1} {
		b.set(hull, 0)
		b.box(v3(sx*5.0, 0, 0.5), v3(1.2, 0.6, 1.6))
		b.set(plate, 0)
		b.box(v3(sx*6.8, 0, 0), v3(1.25, 1.7, 4.2))
		b.set(glow, 1)
		b.box(v3(sx*6.8, 0, -4.25), v3(0.8, 1.1, 0.12))
		b.box(v3(sx*8.1, 0, 0), v3(0.08, 0.3, 3.2))
		b.box(v3(sx*1.6, 0, 6.55), v3(0.9, 0.9, 0.12))
	}
	return b.v
}

// Boss: a winged dreadnought.
func buildBoss() []float32 {
	b := &MB{}
	glow := v3(0.25, 1.0, 0.40)
	hull := v3(0.11, 0.14, 0.15)
	plate := v3(0.20, 0.25, 0.26)
	up := v3(0, 1, 0)
	b.set(hull, 0)
	b.loft([]sec{
		{z: -15, w: 0.8, h: 0.6},
		{z: -8, w: 4.0, h: 2.5},
		{z: 4, w: 6.0, h: 3.6},
		{z: 13, w: 4.0, h: 2.6},
	}, 8)
	b.set(plate, 0)
	b.loft([]sec{
		{z: -3, w: 1.2, h: 0.6, y: 3.2},
		{z: 0, w: 2.4, h: 1.6, y: 4.0},
		{z: 6, w: 2.0, h: 1.2, y: 3.8},
	}, 6)
	b.set(glow, 1)
	b.octa(v3(0, 0, -11.5), v3(1.5, 1.1, 2.6))
	b.box(v3(0, 5.0, 0.2), v3(1.4, 0.25, 0.9))
	for _, sx := range []float32{-1, 1} {
		b.set(hull, 0)
		b.plate(v3(sx*5, 0, -5), v3(sx*5, 0, 10), v3(sx*21, 2.5, 13), v3(sx*21, 2.5, 3), up, 1.8, 0.4)
		b.plate(v3(sx*3, -2, 2), v3(sx*3, -2, 10), v3(sx*8, -8, 13), v3(sx*8, -8, 8), v3(1, 0, 0), 0.9, 0.3)
		b.set(plate, 0)
		b.loft([]sec{
			{z: -9, w: 0.2, h: 0.2, x: sx * 21, y: 2.5},
			{z: -1, w: 1.2, h: 1.2, x: sx * 21, y: 2.5},
			{z: 12, w: 1.4, h: 1.4, x: sx * 21, y: 2.5},
		}, 6)
		b.set(glow, 1)
		b.plate(v3(sx*6, 1.05, 6.5), v3(sx*6, 1.05, 7.5), v3(sx*20, 2.85, 10.5), v3(sx*20, 2.85, 9.5), up, 0.1, 0.1)
		b.plate(v3(sx*6, 1.05, 1.5), v3(sx*6, 1.05, 2.3), v3(sx*20, 2.85, 5.5), v3(sx*20, 2.85, 4.9), up, 0.1, 0.1)
		b.box(v3(sx*21, 2.5, -1.2), v3(0.5, 0.5, 0.4))
		b.loft([]sec{
			{z: 13, w: 1.3, h: 1.3, x: sx * 2.2},
			{z: 13.6, w: 0.9, h: 0.9, x: sx * 2.2},
		}, 8)
		b.loft([]sec{
			{z: 12, w: 1.0, h: 1.0, x: sx * 21, y: 2.5},
			{z: 12.5, w: 0.7, h: 0.7, x: sx * 21, y: 2.5},
		}, 8)
	}
	return b.v
}

func buildPickup() []float32 {
	b := &MB{}
	b.set(v3(1, 1, 1), 1)
	b.octa(v3(0, 0, 0), v3(1, 1.5, 1))
	return b.v
}

func buildAsteroid(seed float32) []float32 {
	b := &MB{}
	off := v3(seed*13.1, seed*7.7, seed*3.3)
	disp := func(p V3) V3 {
		n := fbm(p.Mul(1.4).Add(off), 4)
		r := fbm(p.Mul(3.5).Add(off), 3)
		return p.Mul(0.62 + 0.62*n + 0.18*r)
	}
	for _, t := range icosphere(2) {
		p0, p1, p2 := disp(t[0]), disp(t[1]), disp(t[2])
		k := 0.75 + 0.5*vnoise(p0.Mul(2.5).Add(off))
		b.set(v3(0.30, 0.27, 0.25).Mul(k), 0)
		b.c = V3{}
		b.tri(p0, p1, p2)
	}
	return b.v
}

// buildPlanet makes a smooth-shaded banded gas giant.
func buildPlanet() []float32 {
	const segU, segV = 96, 64
	var out []float32
	pt := func(i, j int) V3 {
		th := math.Pi * float64(j) / segV
		ph := 2 * math.Pi * float64(i) / segU
		return v3(float32(math.Sin(th)*math.Cos(ph)), float32(math.Cos(th)), float32(math.Sin(th)*math.Sin(ph)))
	}
	colAt := func(p V3) V3 {
		n := fbm(p.Mul(3).Add(v3(4, 1, 7)), 4)
		tb := fbm(v3(p[0]*5, p[1]*16, p[2]*5), 3)
		t := float32(math.Sin(float64(p[1]*15+n*5+tb*2)))*0.5 + 0.5
		a, c := v3(0.62, 0.30, 0.16), v3(0.92, 0.78, 0.60)
		col := a.Add(c.Sub(a).Mul(t))
		st := fbm(p.Mul(7).Add(v3(9, 9, 9)), 3)
		if st > 0.62 {
			col = col.Add(v3(0.20, 0.10, 0.25).Sub(col).Mul((st - 0.62) * 4))
		}
		return col
	}
	add := func(p V3) {
		c := colAt(p)
		out = append(out, p[0], p[1], p[2], p[0], p[1], p[2], c[0], c[1], c[2], 0)
	}
	for j := 0; j < segV; j++ {
		for i := 0; i < segU; i++ {
			a, b, c, d := pt(i, j), pt(i+1, j), pt(i+1, j+1), pt(i, j+1)
			for _, t := range [][3]V3{{a, b, c}, {a, c, d}} {
				// keep counter-clockwise winding seen from outside
				if t[1].Sub(t[0]).Cross(t[2].Sub(t[0])).Dot(t[0].Add(t[1]).Add(t[2])) < 0 {
					t[1], t[2] = t[2], t[1]
				}
				add(t[0])
				add(t[1])
				add(t[2])
			}
		}
	}
	return out
}

// Rocket: a slim body with four fins and a glowing tail.
func buildRocket() []float32 {
	b := &MB{}
	b.set(v3(0.85, 0.87, 0.92), 0)
	b.loft([]sec{
		{z: -2.4, w: 0.04, h: 0.04},
		{z: -1.3, w: 0.36, h: 0.36},
		{z: 1.6, w: 0.36, h: 0.36},
		{z: 1.9, w: 0.26, h: 0.26},
	}, 6)
	b.set(v3(0.9, 0.2, 0.15), 0)
	for i := 0; i < 4; i++ {
		a := float64(i) * math.Pi / 2
		d := v3(float32(math.Cos(a)), float32(math.Sin(a)), 0)
		t := v3(-d[1], d[0], 0)
		z := func(r, zz float32) V3 { return d.Mul(r).Add(v3(0, 0, zz)) }
		b.plate(z(0.3, 0.6), z(0.3, 1.8), z(1.1, 2.1), z(1.1, 1.5), t, 0.1, 0.05)
	}
	b.set(v3(1.0, 0.6, 0.2), 1)
	b.octa(v3(0, 0, 2.0), v3(0.22, 0.22, 0.35))
	return b.v
}
