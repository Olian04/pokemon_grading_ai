package openai

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestImageDataURLEmpty(t *testing.T) {
	_, err := imageDataURL(nil)
	if !errors.Is(err, ErrEmptyImageBytes) {
		t.Fatalf("got %v want ErrEmptyImageBytes", err)
	}
	_, err = imageDataURL([]byte{})
	if !errors.Is(err, ErrEmptyImageBytes) {
		t.Fatalf("got %v want ErrEmptyImageBytes", err)
	}
}

func TestImageDataURLUnsupported(t *testing.T) {
	_, err := imageDataURL([]byte("not an image"))
	if !errors.Is(err, ErrUnsupportedImageType) {
		t.Fatalf("got %v want ErrUnsupportedImageType", err)
	}
}

func TestImageDataURLPNG(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 10, G: 20, B: 30, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	u, err := imageDataURL(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(u, "data:image/png;base64,") {
		prefixLen := 40
		if len(u) < prefixLen {
			prefixLen = len(u)
		}
		t.Fatalf("unexpected prefix: %q", u[:prefixLen])
	}
}
