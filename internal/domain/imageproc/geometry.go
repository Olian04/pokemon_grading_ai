package imageproc

import "math"

// Pt is a floating-point 2D point in pixel coordinates.
type Pt struct {
	X, Y float64
}

func (p Pt) add(q Pt) Pt      { return Pt{p.X + q.X, p.Y + q.Y} }
func (p Pt) sub(q Pt) Pt      { return Pt{p.X - q.X, p.Y - q.Y} }
func (p Pt) mul(s float64) Pt { return Pt{p.X * s, p.Y * s} }
func (p Pt) dot(q Pt) float64 { return p.X*q.X + p.Y*q.Y }
func (p Pt) len() float64     { return math.Hypot(p.X, p.Y) }

// cross2 returns z component of (a-b) x (c-b) in 2D (positive if c is left of line ab).
func cross2(a, b, c Pt) float64 {
	return (c.X-b.X)*(a.Y-b.Y) - (c.Y-b.Y)*(a.X-b.X)
}

// convexQuad returns true if pts form a convex quadrilateral in order.
func convexQuad(pts []Pt) bool {
	if len(pts) != 4 {
		return false
	}
	var signs []float64
	n := len(pts)
	for i := 0; i < n; i++ {
		a := pts[i]
		b := pts[(i+1)%n]
		c := pts[(i+2)%n]
		z := cross2(a, b, c)
		if math.Abs(z) < 1e-9 {
			continue
		}
		signs = append(signs, math.Copysign(1, z))
	}
	if len(signs) == 0 {
		return false
	}
	first := signs[0]
	for _, s := range signs[1:] {
		if s != first {
			return false
		}
	}
	return true
}

// quadAspectRatio returns width/height of the min bounding rect of the quad (axis-aligned bbox of corners).
func quadAspectRatio(corners []Pt) float64 {
	if len(corners) != 4 {
		return 0
	}
	minX, maxX := corners[0].X, corners[0].X
	minY, maxY := corners[0].Y, corners[0].Y
	for _, p := range corners[1:] {
		minX = math.Min(minX, p.X)
		maxX = math.Max(maxX, p.X)
		minY = math.Min(minY, p.Y)
		maxY = math.Max(maxY, p.Y)
	}
	w := maxX - minX
	h := maxY - minY
	if h < 1e-6 {
		return 0
	}
	return w / h
}

// quadArea returns polygon area (shoelace); corners must be ordered (cw or ccw).
func quadArea(corners []Pt) float64 {
	if len(corners) != 4 {
		return 0
	}
	var a float64
	for i := 0; i < 4; i++ {
		j := (i + 1) % 4
		a += corners[i].X*corners[j].Y - corners[j].X*corners[i].Y
	}
	return math.Abs(a) / 2
}

// orderCornersTLTRBRBL sorts four corners into top-left, top-right, bottom-right, bottom-left
// for a convex quadrilateral viewed roughly from above (card face).
func orderCornersTLTRBRBL(corners []Pt) []Pt {
	if len(corners) != 4 {
		return nil
	}
	var cx, cy float64
	for _, p := range corners {
		cx += p.X
		cy += p.Y
	}
	cx /= 4
	cy /= 4
	return orderByAngle(corners, cx, cy)
}

func orderByAngle(corners []Pt, cx, cy float64) []Pt {
	type ang struct {
		p Pt
		a float64
	}
	var as []ang
	for _, p := range corners {
		a := math.Atan2(p.Y-cy, p.X-cx)
		as = append(as, ang{p: p, a: a})
	}
	// sort by angle ascending
	for i := 0; i < len(as); i++ {
		for j := i + 1; j < len(as); j++ {
			if as[j].a < as[i].a {
				as[i], as[j] = as[j], as[i]
			}
		}
	}
	// Find starting index with smallest y (top-most), tie smallest x
	start := 0
	for i := 1; i < 4; i++ {
		if as[i].p.Y < as[start].p.Y-1e-6 || (math.Abs(as[i].p.Y-as[start].p.Y) < 1e-6 && as[i].p.X < as[start].p.X) {
			start = i
		}
	}
	out := make([]Pt, 4)
	for i := 0; i < 4; i++ {
		out[i] = as[(start+i)%4].p
	}
	// ang sort gives TL at start for typical card if start is top-left on boundary — may need rotate so TL has min x+y
	// rotate so that vertex with min (x+y) is first
	best := 0
	bestS := out[0].X + out[0].Y
	for i := 1; i < 4; i++ {
		s := out[i].X + out[i].Y
		if s < bestS {
			bestS = s
			best = i
		}
	}
	rot := make([]Pt, 4)
	for i := 0; i < 4; i++ {
		rot[i] = out[(best+i)%4]
	}
	return rot
}

// pcaOBBCorners builds four corners of the oriented bounding box in PCA space of hull points
// (approximation to a card silhouette); returns TL,TR,BR,BL in screen order for a convex quad.
func pcaOBBCorners(hull []Pt) []Pt {
	if len(hull) < 3 {
		return nil
	}
	var cx, cy float64
	for _, p := range hull {
		cx += p.X
		cy += p.Y
	}
	n := float64(len(hull))
	cx /= n
	cy /= n
	var sxx, syy, sxy float64
	for _, p := range hull {
		dx := p.X - cx
		dy := p.Y - cy
		sxx += dx * dx
		syy += dy * dy
		sxy += dx * dy
	}
	sxx /= n
	syy /= n
	sxy /= n
	// Eigenvectors of 2x2 covariance
	trace := sxx + syy
	det := sxx*syy - sxy*sxy
	disc := trace*trace/4 - det
	if disc < 0 {
		disc = 0
	}
	l1 := trace/2 + math.Sqrt(disc)
	var ex, ey float64
	if math.Abs(sxy) > 1e-12 {
		ex, ey = l1-syy, sxy
	} else if sxx >= syy {
		ex, ey = 1, 0
	} else {
		ex, ey = 0, 1
	}
	n1 := math.Hypot(ex, ey)
	if n1 < 1e-12 {
		return nil
	}
	e1 := Pt{ex / n1, ey / n1}
	e2 := Pt{-e1.Y, e1.X}

	var umin, umax, vmin, vmax float64
	first := true
	for _, p := range hull {
		d := p.sub(Pt{cx, cy})
		u := d.dot(e1)
		v := d.dot(e2)
		if first {
			umin, umax = u, u
			vmin, vmax = v, v
			first = false
			continue
		}
		umin = math.Min(umin, u)
		umax = math.Max(umax, u)
		vmin = math.Min(vmin, v)
		vmax = math.Max(vmax, v)
	}
	// Expand slightly so quad contains silhouette (helps thin masks)
	padU := (umax - umin) * 0.02
	padV := (vmax - vmin) * 0.02
	umin -= padU
	umax += padU
	vmin -= padV
	vmax += padV

	c := Pt{cx, cy}
	raw := []Pt{
		c.add(e1.mul(umin)).add(e2.mul(vmin)),
		c.add(e1.mul(umax)).add(e2.mul(vmin)),
		c.add(e1.mul(umax)).add(e2.mul(vmax)),
		c.add(e1.mul(umin)).add(e2.mul(vmax)),
	}
	return orderCornersTLTRBRBL(raw)
}
