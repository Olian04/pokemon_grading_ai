package imageproc

import (
	"image"
	"image/color"
	"math"

	"golang.org/x/image/draw"
)

// QuadMeta summarizes the last normalization attempt (for evidence / debugging).
type QuadMeta struct {
	AreaRatio    float64
	FallbackUsed bool
}

// NormalizeCard detects a card silhouette, estimates four corners, dewarps to a frontal
// rectangle with Pokémon aspect ratio (63:88), and returns RGBA suitable for heuristics.
// Corners are estimated in working resolution then mapped to orig for sampling.
func NormalizeCard(orig *image.RGBA, cfg Config, dbg *DebugSink) (*image.RGBA, QuadMeta, error) {
	var meta QuadMeta
	_ = dbg.Write("01_decode", orig)

	work := resizeMaxLongEdge(orig, cfg.MaxWorkingLongEdge)
	_ = dbg.Write("02_downscale", work)

	ob := orig.Bounds()
	wb := work.Bounds()
	sx := float64(ob.Dx()) / float64(wb.Dx())
	sy := float64(ob.Dy()) / float64(wb.Dy())
	offX := float64(ob.Min.X)
	offY := float64(ob.Min.Y)

	g := grayscale(work)
	_ = dbg.Write("03_preprocess_gray", rgbaFromGray(g))
	g = simpleBlurGray(g, 2)
	_ = dbg.Write("04_blur", rgbaFromGray(g))

	thr := otsuThreshold(g)
	maskBright := largestComponentMask(dilateMask(binarize(g, thr, true), 2), 0.93)
	maskDark := largestComponentMask(dilateMask(binarize(g, thr, false), 2), 0.93)
	areaB := componentArea(maskBright)
	areaD := componentArea(maskDark)
	var mask [][]uint8
	frameBright := areaB > 0 && maskTouchesAllImageEdges(maskBright) &&
		float64(areaB)/float64(wb.Dx()*wb.Dy()) > 0.22
	switch {
	case areaB == 0 && areaD == 0:
		return nil, meta, ErrNoCardQuad
	case frameBright && areaD > 0:
		// Bright foreground is usually the desk/window frame; the card is the inner dark blob.
		mask = maskDark
	case areaB >= areaD && areaB > 0:
		mask = maskBright
	case areaD > 0:
		mask = maskDark
	default:
		mask = maskBright
	}
	_ = dbg.Write("05_mask", maskToRGBA(mask))

	pts := boundaryPoints(mask)
	if len(pts) < 12 {
		return nil, meta, ErrNoCardQuad
	}
	hull := convexHull(pts)
	if len(hull) < 4 {
		return nil, meta, ErrNoCardQuad
	}
	cornersWork := pcaOBBCorners(hull)
	if cornersWork == nil || len(cornersWork) != 4 {
		return nil, meta, ErrNoCardQuad
	}
	// Winding may be CW or CCW depending on corner order; require non-degenerate area.
	if quadArea(cornersWork) < 50 {
		return nil, meta, ErrNoCardQuad
	}
	ar := quadArea(cornersWork) / float64(wb.Dx()*wb.Dy())
	meta.AreaRatio = ar
	if ar < cfg.MinQuadAreaRatio || ar > cfg.MaxQuadAreaRatio {
		return nil, meta, ErrNoCardQuad
	}
	aspect := quadAspectRatio(cornersWork)
	// Silhouette OBB aspect can differ from physical 63:88 when the mask is partial; keep a wide band.
	if aspect < 0.35 || aspect > 1.3 {
		return nil, meta, ErrNoCardQuad
	}

	_ = dbg.Write("06_quad_overlay", drawQuadOnRGBA(work, cornersWork, color.RGBA{R: 255, A: 255}))
	_ = dbg.Write("07_ordered_corners", drawQuadCornersOnRGBA(work, cornersWork))

	srcOrig := make([]Pt, 4)
	for i, p := range cornersWork {
		srcOrig[i] = Pt{p.X*sx + offX, p.Y*sy + offY}
	}

	wOut := cfg.WarpWidth
	if wOut < 64 {
		wOut = 800
	}
	hOut := int(math.Round(float64(wOut) * 88 / 63))
	dst := []Pt{
		{0, 0},
		{float64(wOut - 1), 0},
		{float64(wOut - 1), float64(hOut - 1)},
		{0, float64(hOut - 1)},
	}

	H, ok := homographyFrom4Points(srcOrig, dst)
	if !ok {
		return nil, meta, ErrDegenerateHomography
	}
	invH, ok := invert3x3(H)
	if !ok {
		return nil, meta, ErrDegenerateHomography
	}
	out := warpPerspectiveInverse(orig, invH, wOut, hOut)
	_ = dbg.Write("08_warped", out)
	_ = dbg.Write("09_final", out)
	return out, meta, nil
}

// maskTouchesAllImageEdges is true when foreground (1) appears on all four borders of the bitmap.
func maskTouchesAllImageEdges(m [][]uint8) bool {
	h, w := len(m), len(m[0])
	var top, bot, left, right bool
	for x := 0; x < w; x++ {
		if m[0][x] == 1 {
			top = true
		}
		if m[h-1][x] == 1 {
			bot = true
		}
	}
	for y := 0; y < h; y++ {
		if m[y][0] == 1 {
			left = true
		}
		if m[y][w-1] == 1 {
			right = true
		}
	}
	return top && bot && left && right
}

func componentArea(m [][]uint8) int {
	h, w := len(m), len(m[0])
	n := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if m[y][x] == 1 {
				n++
			}
		}
	}
	return n
}

func drawQuadOnRGBA(src *image.RGBA, q []Pt, col color.RGBA) *image.RGBA {
	b := src.Bounds()
	out := image.NewRGBA(b)
	draw.Draw(out, b, src, b.Min, draw.Src)
	if len(q) != 4 {
		return out
	}
	for i := 0; i < 4; i++ {
		a := q[i]
		bb := q[(i+1)%4]
		lineRGBA(out, int(a.X+0.5), int(a.Y+0.5), int(bb.X+0.5), int(bb.Y+0.5), col)
	}
	return out
}

func drawQuadCornersOnRGBA(src *image.RGBA, q []Pt) *image.RGBA {
	b := src.Bounds()
	out := image.NewRGBA(b)
	draw.Draw(out, b, src, b.Min, draw.Src)
	labels := []color.RGBA{
		{255, 0, 0, 255},
		{0, 255, 0, 255},
		{0, 0, 255, 255},
		{255, 255, 0, 255},
	}
	for i, p := range q {
		x, y := int(p.X+0.5), int(p.Y+0.5)
		for dy := -3; dy <= 3; dy++ {
			for dx := -3; dx <= 3; dx++ {
				xx, yy := x+dx, y+dy
				if xx >= b.Min.X && xx < b.Max.X && yy >= b.Min.Y && yy < b.Max.Y {
					out.SetRGBA(xx, yy, labels[i%4])
				}
			}
		}
	}
	return out
}

func lineRGBA(img *image.RGBA, x0, y0, x1, y1 int, col color.RGBA) {
	dx := absInt(x1 - x0)
	dy := absInt(y1 - y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx - dy
	x, y := x0, y0
	b := img.Bounds()
	for {
		if x >= b.Min.X && x < b.Max.X && y >= b.Min.Y && y < b.Max.Y {
			img.SetRGBA(x, y, col)
		}
		if x == x1 && y == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x += sx
		}
		if e2 < dx {
			err += dx
			y += sy
		}
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
