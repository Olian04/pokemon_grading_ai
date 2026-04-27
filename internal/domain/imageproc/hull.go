package imageproc

import (
	"math"
	"sort"
)

// boundaryPoints collects (x,y) of foreground pixels adjacent to background (4-neighbor).
func boundaryPoints(m [][]uint8) []Pt {
	h, w := len(m), len(m[0])
	var pts []Pt
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if m[y][x] == 0 {
				continue
			}
			touch := y == 0 || y == h-1 || x == 0 || x == w-1
			if !touch {
				touch = m[y-1][x] == 0 || m[y+1][x] == 0 || m[y][x-1] == 0 || m[y][x+1] == 0
			}
			if touch {
				pts = append(pts, Pt{float64(x), float64(y)})
			}
		}
	}
	// subsample if huge
	if len(pts) > 4000 {
		step := len(pts) / 2000
		if step < 1 {
			step = 1
		}
		var sparse []Pt
		for i := 0; i < len(pts); i += step {
			sparse = append(sparse, pts[i])
		}
		pts = sparse
	}
	return pts
}

// convexHull returns vertices of convex hull in CCW order (Graham scan).
func convexHull(points []Pt) []Pt {
	if len(points) < 3 {
		return nil
	}
	// lowest then leftmost point
	start := 0
	for i := 1; i < len(points); i++ {
		if points[i].Y < points[start].Y-1e-9 || (absFloat(points[i].Y-points[start].Y) < 1e-9 && points[i].X < points[start].X) {
			start = i
		}
	}
	o := points[start]
	rest := make([]Pt, 0, len(points)-1)
	for i, p := range points {
		if i == start {
			continue
		}
		rest = append(rest, p)
	}
	sort.Slice(rest, func(i, j int) bool {
		ai := math.Atan2(rest[i].Y-o.Y, rest[i].X-o.X)
		aj := math.Atan2(rest[j].Y-o.Y, rest[j].X-o.X)
		if absFloat(ai-aj) < 1e-12 {
			return dist2(rest[i], o) < dist2(rest[j], o)
		}
		return ai < aj
	})
	hull := []Pt{o}
	for _, p := range rest {
		for len(hull) >= 2 && cross2(hull[len(hull)-2], hull[len(hull)-1], p) <= 0 {
			hull = hull[:len(hull)-1]
		}
		hull = append(hull, p)
	}
	return hull
}

func dist2(a, b Pt) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return dx*dx + dy*dy
}
