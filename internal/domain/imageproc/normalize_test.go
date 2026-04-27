package imageproc

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"math"
	"testing"
)

func TestNormalizeCardAxisAlignedDarkCard(t *testing.T) {
	w, h := 320, 260
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 252, G: 252, B: 252, A: 255})
		}
	}
	for y := 55; y < 210; y++ {
		for x := 70; x < 250; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 15, G: 15, B: 15, A: 255})
		}
	}

	cfg := DefaultConfig()
	cfg.CardNormalize = true
	cfg.StrictCardNormalize = true
	cfg.MinQuadAreaRatio = 0.08
	cfg.MaxQuadAreaRatio = 0.95

	dbg := &DebugSink{}
	out, _, err := NormalizeCard(img, cfg, dbg)
	if err != nil {
		t.Fatalf("NormalizeCard: %v", err)
	}
	if out == nil {
		t.Fatal("nil output")
	}
	br := out.Bounds()
	if br.Dx() != cfg.WarpWidth {
		t.Fatalf("warp width got %d want %d", br.Dx(), cfg.WarpWidth)
	}
	wantH := int(math.Round(float64(cfg.WarpWidth) * 88 / 63))
	if br.Dy() != wantH {
		t.Fatalf("warp height got %d want %d", br.Dy(), wantH)
	}
}

func TestNormalizeCardUniformFieldNoQuad(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 220, 220))
	for y := 0; y < 220; y++ {
		for x := 0; x < 220; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 200, G: 200, B: 200, A: 255})
		}
	}
	cfg := DefaultConfig()
	cfg.CardNormalize = true
	_, _, err := NormalizeCard(img, cfg, &DebugSink{})
	if !errors.Is(err, ErrNoCardQuad) {
		t.Fatalf("got %v want ErrNoCardQuad", err)
	}
}

func BenchmarkNormalizeCardDownscaled(b *testing.B) {
	w, h := 1280, 960
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 250, G: 250, B: 250, A: 255})
		}
	}
	for y := 200; y < 760; y++ {
		for x := 300; x < 980; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 20, G: 20, B: 20, A: 255})
		}
	}
	cfg := DefaultConfig()
	cfg.MaxWorkingLongEdge = 960
	cfg.WarpWidth = 800
	dbg := &DebugSink{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := NormalizeCard(img, cfg, dbg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestPcaOBBCornersOnSquare(t *testing.T) {
	hull := []Pt{{0, 0}, {10, 0}, {10, 10}, {0, 10}}
	q := pcaOBBCorners(hull)
	if len(q) != 4 {
		t.Fatalf("got %d corners", len(q))
	}
	if !convexQuad(q) {
		t.Fatal("expected convex quad")
	}
}

func TestDebugSinkWritesSteps(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	cfg := DefaultConfig()
	cfg.DebugNormalize = true
	cfg.DebugNormalizeOutDir = tmp
	dbg, err := newDebugSink(cfg, "test")
	if err != nil {
		t.Fatal(err)
	}
	im := image.NewRGBA(image.Rect(0, 0, 10, 10))
	im.SetRGBA(1, 1, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	if err := dbg.Write("sample", im); err != nil {
		t.Fatal(err)
	}
	if dbg.dir() == "" {
		t.Fatal("expected debug dir")
	}
}

func TestAnalyzeWithStrictNormalizeFailsOnBlank(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CardNormalize = true
	cfg.StrictCardNormalize = true
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 200, G: 200, B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	_, err := NewAnalyzer(cfg).Analyze(buf.Bytes())
	if err == nil {
		t.Fatal("expected error for strict normalize on uniform field")
	}
}
