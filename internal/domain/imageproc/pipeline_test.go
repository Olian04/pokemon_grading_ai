package imageproc

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestAnalyzeValidImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 120, 120))
	for y := 0; y < 120; y++ {
		for x := 0; x < 120; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 40, G: 80, B: 120, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	a := NewDefaultAnalyzer()
	got, err := a.Analyze(buf.Bytes())
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if got.CenteringScore < 1 || got.CenteringScore > 10 {
		t.Fatalf("centering out of range: %v", got.CenteringScore)
	}
	if len(got.Evidence) == 0 {
		t.Fatal("expected evidence strings")
	}
}

func TestAnalyzeEmptyBytes(t *testing.T) {
	_, err := NewDefaultAnalyzer().Analyze(nil)
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestAnalyzeTooSmall(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	_, err := NewDefaultAnalyzer().Analyze(buf.Bytes())
	if err == nil {
		t.Fatal("expected error for small image")
	}
}
