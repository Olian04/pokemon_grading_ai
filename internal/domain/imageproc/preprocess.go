package imageproc

import (
	"image"
	"image/color"
	"math"

	"golang.org/x/image/draw"
)

func toRGBA(img image.Image) *image.RGBA {
	b := img.Bounds()
	out := image.NewRGBA(b)
	draw.Draw(out, b, img, b.Min, draw.Src)
	return out
}

func resizeMaxLongEdge(src *image.RGBA, maxLong int) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	long := maxInt(w, h)
	if long <= maxLong {
		return src
	}
	scale := float64(maxLong) / float64(long)
	nw := maxInt(1, int(math.Round(float64(w)*scale)))
	nh := maxInt(1, int(math.Round(float64(h)*scale)))
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

func grayscale(src *image.RGBA) [][]float64 {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	g := make([][]float64, h)
	for y := 0; y < h; y++ {
		g[y] = make([]float64, w)
		for x := 0; x < w; x++ {
			g[y][x] = pixelLumaRGBA(src, b.Min.X+x, b.Min.Y+y)
		}
	}
	return g
}

func pixelLumaRGBA(img *image.RGBA, x, y int) float64 {
	r, g, b, _ := img.At(x, y).RGBA()
	rf := float64(r >> 8)
	gf := float64(g >> 8)
	bf := float64(b >> 8)
	return 0.2126*rf + 0.7152*gf + 0.0722*bf
}

func simpleBlurGray(g [][]float64, passes int) [][]float64 {
	h, w := len(g), len(g[0])
	cur := g
	for p := 0; p < passes; p++ {
		out := make([][]float64, h)
		for y := 0; y < h; y++ {
			out[y] = make([]float64, w)
		}
		for y := 1; y < h-1; y++ {
			for x := 1; x < w-1; x++ {
				var s float64
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						s += cur[y+dy][x+dx]
					}
				}
				out[y][x] = s / 9
			}
		}
		// copy borders
		for x := 0; x < w; x++ {
			out[0][x] = cur[0][x]
			out[h-1][x] = cur[h-1][x]
		}
		for y := 0; y < h; y++ {
			out[y][0] = cur[y][0]
			out[y][w-1] = cur[y][w-1]
		}
		cur = out
	}
	return cur
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func otsuThreshold(g [][]float64) int {
	h, w := len(g), len(g[0])
	hist := [256]int{}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := int(g[y][x] + 0.5)
			if v < 0 {
				v = 0
			}
			if v > 255 {
				v = 255
			}
			hist[v]++
		}
	}
	total := w * h
	var sum float64
	for i := 0; i < 256; i++ {
		sum += float64(i * hist[i])
	}
	var sumB float64
	var wB, wF int
	var maxVar float64
	threshold := 0
	for t := 0; t < 256; t++ {
		wB += hist[t]
		if wB == 0 {
			continue
		}
		wF = total - wB
		if wF == 0 {
			break
		}
		sumB += float64(t * hist[t])
		mB := sumB / float64(wB)
		mF := (sum - sumB) / float64(wF)
		between := float64(wB) * float64(wF) * (mB - mF) * (mB - mF)
		if between > maxVar {
			maxVar = between
			threshold = t
		}
	}
	return threshold
}

func binarize(g [][]float64, thr int, foregroundBright bool) [][]uint8 {
	h, w := len(g), len(g[0])
	out := make([][]uint8, h)
	for y := 0; y < h; y++ {
		out[y] = make([]uint8, w)
		for x := 0; x < w; x++ {
			v := int(g[y][x] + 0.5)
			if foregroundBright {
				if v >= thr {
					out[y][x] = 1
				}
			} else {
				if v <= thr {
					out[y][x] = 1
				}
			}
		}
	}
	return out
}

func invertMask(m [][]uint8) [][]uint8 {
	h, w := len(m), len(m[0])
	out := make([][]uint8, h)
	for y := 0; y < h; y++ {
		out[y] = make([]uint8, w)
		for x := 0; x < w; x++ {
			out[y][x] = 1 - m[y][x]
		}
	}
	return out
}

func dilateMask(m [][]uint8, iters int) [][]uint8 {
	h, w := len(m), len(m[0])
	cur := m
	for k := 0; k < iters; k++ {
		next := make([][]uint8, h)
		for y := 0; y < h; y++ {
			next[y] = make([]uint8, w)
			copy(next[y], cur[y])
		}
		for y := 1; y < h-1; y++ {
			for x := 1; x < w-1; x++ {
				if cur[y][x] == 1 {
					continue
				}
				found := false
				for dy := -1; dy <= 1 && !found; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if cur[y+dy][x+dx] == 1 {
							found = true
							break
						}
					}
				}
				if found {
					next[y][x] = 1
				}
			}
		}
		cur = next
	}
	return cur
}

func rgbaFromGray(g [][]float64) *image.RGBA {
	h, w := len(g), len(g[0])
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := uint8(clampInt(int(g[y][x]+0.5), 0, 255))
			out.SetRGBA(x, y, color.RGBA{R: v, G: v, B: v, A: 255})
		}
	}
	return out
}

func maskToRGBA(m [][]uint8) *image.RGBA {
	h, w := len(m), len(m[0])
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := uint8(0)
			if m[y][x] == 1 {
				v = 255
			}
			out.SetRGBA(x, y, color.RGBA{R: v, G: v, B: v, A: 255})
		}
	}
	return out
}
