package main

import (
	"math"

	mgl "github.com/go-gl/mathgl/mgl32"
)

type V3 = mgl.Vec3

func v3(x, y, z float32) V3 { return V3{x, y, z} }

func norm(v V3) V3 {
	l := v.Len()
	if l < 1e-6 {
		return V3{0, 0, -1}
	}
	return v.Mul(1 / l)
}

// MB builds flat-shaded meshes. Vertex layout: pos3, normal3, rgb3, emissive1.
// Triangles are wound and lit facing away from the current part centre `c`.
type MB struct {
	v   []float32
	c   V3
	col V3
	em  float32
}

func (b *MB) set(col V3, em float32) { b.col, b.em = col, em }

func (b *MB) push(p, n V3) {
	b.v = append(b.v, p[0], p[1], p[2], n[0], n[1], n[2], b.col[0], b.col[1], b.col[2], b.em)
}

func (b *MB) tri(p0, p1, p2 V3) {
	n := p1.Sub(p0).Cross(p2.Sub(p0))
	if n.Len() < 1e-9 {
		return
	}
	n = n.Normalize()
	cen := p0.Add(p1).Add(p2).Mul(1.0 / 3.0)
	if n.Dot(cen.Sub(b.c)) < 0 {
		p1, p2 = p2, p1
		n = n.Mul(-1)
	}
	b.push(p0, n)
	b.push(p1, n)
	b.push(p2, n)
}

func (b *MB) quad(p0, p1, p2, p3 V3) {
	b.tri(p0, p1, p2)
	b.tri(p0, p2, p3)
}

// sec is one cross-section of a lofted body: an ellipse-ish N-gon at depth z.
type sec struct{ z, w, h, x, y float32 }

func (b *MB) loft(secs []sec, sides int) {
	ring := func(s sec) []V3 {
		pts := make([]V3, sides)
		for k := 0; k < sides; k++ {
			a := 2 * math.Pi * float64(k) / float64(sides)
			pts[k] = v3(s.x+s.w*float32(math.Cos(a)), s.y+s.h*float32(math.Sin(a)), s.z)
		}
		return pts
	}
	cen := func(s sec) V3 { return v3(s.x, s.y, s.z) }
	for i := 0; i < len(secs)-1; i++ {
		a, c := ring(secs[i]), ring(secs[i+1])
		b.c = cen(secs[i]).Add(cen(secs[i+1])).Mul(0.5)
		for k := 0; k < sides; k++ {
			k2 := (k + 1) % sides
			b.quad(a[k], a[k2], c[k2], c[k])
		}
	}
	cap := func(s sec, inner V3) {
		r := ring(s)
		b.c = inner
		for k := 0; k < sides; k++ {
			b.tri(cen(s), r[k], r[(k+1)%sides])
		}
	}
	n := len(secs)
	cap(secs[0], cen(secs[0]).Add(cen(secs[1])).Mul(0.5))
	cap(secs[n-1], cen(secs[n-1]).Add(cen(secs[n-2])).Mul(0.5))
}

// plate is a tapered slab: p0,p1 are the root edge, p2,p3 the tip edge.
func (b *MB) plate(p0, p1, p2, p3, tdir V3, t0, t1 float32) {
	o0, o1 := tdir.Mul(t0*0.5), tdir.Mul(t1*0.5)
	top := [4]V3{p0.Add(o0), p1.Add(o0), p2.Add(o1), p3.Add(o1)}
	bot := [4]V3{p0.Sub(o0), p1.Sub(o0), p2.Sub(o1), p3.Sub(o1)}
	b.c = p0.Add(p1).Add(p2).Add(p3).Mul(0.25)
	b.quad(top[0], top[1], top[2], top[3])
	b.quad(bot[0], bot[1], bot[2], bot[3])
	for k := 0; k < 4; k++ {
		k2 := (k + 1) % 4
		b.quad(top[k], top[k2], bot[k2], bot[k])
	}
}

func (b *MB) box(c, h V3) {
	b.c = c
	p := func(sx, sy, sz float32) V3 { return v3(c[0]+sx*h[0], c[1]+sy*h[1], c[2]+sz*h[2]) }
	b.quad(p(-1, -1, 1), p(1, -1, 1), p(1, 1, 1), p(-1, 1, 1))
	b.quad(p(-1, -1, -1), p(1, -1, -1), p(1, 1, -1), p(-1, 1, -1))
	b.quad(p(-1, -1, -1), p(-1, -1, 1), p(-1, 1, 1), p(-1, 1, -1))
	b.quad(p(1, -1, -1), p(1, -1, 1), p(1, 1, 1), p(1, 1, -1))
	b.quad(p(-1, 1, -1), p(1, 1, -1), p(1, 1, 1), p(-1, 1, 1))
	b.quad(p(-1, -1, -1), p(1, -1, -1), p(1, -1, 1), p(-1, -1, 1))
}

func (b *MB) octa(c, r V3) {
	b.c = c
	px, nx := c.Add(v3(r[0], 0, 0)), c.Sub(v3(r[0], 0, 0))
	py, ny := c.Add(v3(0, r[1], 0)), c.Sub(v3(0, r[1], 0))
	pz, nz := c.Add(v3(0, 0, r[2])), c.Sub(v3(0, 0, r[2]))
	for _, y := range []V3{py, ny} {
		b.tri(px, y, pz)
		b.tri(pz, y, nx)
		b.tri(nx, y, nz)
		b.tri(nz, y, px)
	}
}

// --- CPU noise (for asteroid shapes and planet bands) ---

func hash3i(x, y, z int32) float32 {
	h := uint32(x)*374761393 + uint32(y)*668265263 + uint32(z)*2246822519
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return float32(h&0xffffff) / float32(0xffffff)
}

func vnoise(p V3) float32 {
	fx, fy, fz := math.Floor(float64(p[0])), math.Floor(float64(p[1])), math.Floor(float64(p[2]))
	ix, iy, iz := int32(fx), int32(fy), int32(fz)
	sm := func(t float32) float32 { return t * t * (3 - 2*t) }
	tx, ty, tz := sm(p[0]-float32(fx)), sm(p[1]-float32(fy)), sm(p[2]-float32(fz))
	lerp := func(a, b, t float32) float32 { return a + (b-a)*t }
	h := func(dx, dy, dz int32) float32 { return hash3i(ix+dx, iy+dy, iz+dz) }
	return lerp(
		lerp(lerp(h(0, 0, 0), h(1, 0, 0), tx), lerp(h(0, 1, 0), h(1, 1, 0), tx), ty),
		lerp(lerp(h(0, 0, 1), h(1, 0, 1), tx), lerp(h(0, 1, 1), h(1, 1, 1), tx), ty), tz)
}

func fbm(p V3, oct int) float32 {
	var s, a float32 = 0, 0.5
	for i := 0; i < oct; i++ {
		s += a * vnoise(p)
		p = p.Mul(2.03).Add(v3(1.7, 9.2, 3.1))
		a *= 0.5
	}
	return s
}

// icosphere returns unit-sphere triangles after `sub` subdivisions.
func icosphere(sub int) [][3]V3 {
	t := float32((1 + math.Sqrt(5)) / 2)
	vs := []V3{
		{-1, t, 0}, {1, t, 0}, {-1, -t, 0}, {1, -t, 0},
		{0, -1, t}, {0, 1, t}, {0, -1, -t}, {0, 1, -t},
		{t, 0, -1}, {t, 0, 1}, {-t, 0, -1}, {-t, 0, 1},
	}
	idx := [][3]int{
		{0, 11, 5}, {0, 5, 1}, {0, 1, 7}, {0, 7, 10}, {0, 10, 11},
		{1, 5, 9}, {5, 11, 4}, {11, 10, 2}, {10, 7, 6}, {7, 1, 8},
		{3, 9, 4}, {3, 4, 2}, {3, 2, 6}, {3, 6, 8}, {3, 8, 9},
		{4, 9, 5}, {2, 4, 11}, {6, 2, 10}, {8, 6, 7}, {9, 8, 1},
	}
	tris := make([][3]V3, 0, 20)
	for _, f := range idx {
		tris = append(tris, [3]V3{norm(vs[f[0]]), norm(vs[f[1]]), norm(vs[f[2]])})
	}
	for s := 0; s < sub; s++ {
		next := make([][3]V3, 0, len(tris)*4)
		for _, tr := range tris {
			a, b, c := tr[0], tr[1], tr[2]
			ab, bc, ca := norm(a.Add(b)), norm(b.Add(c)), norm(c.Add(a))
			next = append(next, [3]V3{a, ab, ca}, [3]V3{b, bc, ab}, [3]V3{c, ca, bc}, [3]V3{ab, bc, ca})
		}
		tris = next
	}
	return tris
}
