package imageproc

import (
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
)

type Result struct {
	CenteringScore float64
	CornersScore   float64
	EdgesScore     float64
	SurfaceScore   float64
	Confidence     float64
	Evidence       []string
}

type Analyzer struct{}

func NewAnalyzer() Analyzer {
	return Analyzer{}
}

func (Analyzer) Analyze(path string) (Result, error) {
	if path == "" {
		return Result{}, errors.New("image path is required")
	}
	f, err := os.Open(path)
	if err != nil {
		return Result{}, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return Result{}, err
	}

	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 100 || h < 100 {
		return Result{}, errors.New("insufficient_image_quality: image too small")
	}

	leftMean := edgeLumaMean(img, b.Min.X, b.Min.Y, w/16, h)
	rightMean := edgeLumaMean(img, b.Max.X-w/16, b.Min.Y, w/16, h)
	topMean := edgeLumaMean(img, b.Min.X, b.Min.Y, w, h/16)
	bottomMean := edgeLumaMean(img, b.Min.X, b.Max.Y-h/16, w, h/16)

	centeringX := 10 - minFloat(10, absFloat(leftMean-rightMean)/10)
	centeringY := 10 - minFloat(10, absFloat(topMean-bottomMean)/10)
	centering := maxFloat(1, (centeringX+centeringY)/2)

	cornerContrast := cornerDelta(img, b)
	corners := clampScore(10 - cornerContrast/12)

	edgeNoise := borderNoise(img, b)
	edges := clampScore(10 - edgeNoise/14)

	surfaceVar := centerVariance(img, b)
	surface := clampScore(10 - surfaceVar/18)

	confidence := clampUnit(0.55 + float64(minInt(w, h))/3000)
	return Result{
		CenteringScore: round1(centering),
		CornersScore:   round1(corners),
		EdgesScore:     round1(edges),
		SurfaceScore:   round1(surface),
		Confidence:     round2(confidence),
		Evidence: []string{
			"deterministic border luminance symmetry",
			"deterministic corner contrast heuristic",
			"deterministic edge noise and surface variance",
		},
	}, nil
}

func edgeLumaMean(img image.Image, startX, startY, width, height int) float64 {
	var total float64
	var count int
	for y := startY; y < startY+height; y++ {
		for x := startX; x < startX+width; x++ {
			total += pixelLuma(img, x, y)
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func cornerDelta(img image.Image, b image.Rectangle) float64 {
	cw := maxInt(8, b.Dx()/10)
	ch := maxInt(8, b.Dy()/10)
	lt := edgeLumaMean(img, b.Min.X, b.Min.Y, cw, ch)
	rt := edgeLumaMean(img, b.Max.X-cw, b.Min.Y, cw, ch)
	lb := edgeLumaMean(img, b.Min.X, b.Max.Y-ch, cw, ch)
	rb := edgeLumaMean(img, b.Max.X-cw, b.Max.Y-ch, cw, ch)
	mean := (lt + rt + lb + rb) / 4
	return (absFloat(lt-mean) + absFloat(rt-mean) + absFloat(lb-mean) + absFloat(rb-mean)) / 4
}

func borderNoise(img image.Image, b image.Rectangle) float64 {
	bw := maxInt(4, b.Dx()/30)
	bh := maxInt(4, b.Dy()/30)
	left := stddevLuma(img, b.Min.X, b.Min.Y, bw, b.Dy())
	right := stddevLuma(img, b.Max.X-bw, b.Min.Y, bw, b.Dy())
	top := stddevLuma(img, b.Min.X, b.Min.Y, b.Dx(), bh)
	bottom := stddevLuma(img, b.Min.X, b.Max.Y-bh, b.Dx(), bh)
	return (left + right + top + bottom) / 4
}

func centerVariance(img image.Image, b image.Rectangle) float64 {
	w := b.Dx() / 2
	h := b.Dy() / 2
	return stddevLuma(img, b.Min.X+b.Dx()/4, b.Min.Y+b.Dy()/4, w, h)
}

func stddevLuma(img image.Image, startX, startY, width, height int) float64 {
	var values []float64
	for y := startY; y < startY+height; y++ {
		for x := startX; x < startX+width; x++ {
			values = append(values, pixelLuma(img, x, y))
		}
	}
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))
	var sq float64
	for _, v := range values {
		d := v - mean
		sq += d * d
	}
	return math.Sqrt(sq / float64(len(values)))
}

func pixelLuma(img image.Image, x, y int) float64 {
	r, g, b, _ := img.At(x, y).RGBA()
	rf := float64(r >> 8)
	gf := float64(g >> 8)
	bf := float64(b >> 8)
	return 0.2126*rf + 0.7152*gf + 0.0722*bf
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func clampScore(v float64) float64 {
	if v < 1 {
		return 1
	}
	if v > 10 {
		return 10
	}
	return v
}

func clampUnit(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
