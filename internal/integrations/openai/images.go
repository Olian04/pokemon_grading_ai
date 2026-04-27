package openai

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

// imageDataURL returns an OpenAI-compatible data URL for image_url parts.
func imageDataURL(b []byte) (string, error) {
	if len(b) == 0 {
		return "", ErrEmptyImageBytes
	}
	mime := http.DetectContentType(b)
	if !strings.HasPrefix(mime, "image/") {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedImageType, mime)
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b), nil
}
