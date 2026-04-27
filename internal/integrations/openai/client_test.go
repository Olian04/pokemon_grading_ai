package openai

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pokemon_ai/internal/domain/grading"
)

func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 50, G: 100, B: 150, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestAssessSurfaceRequestMultimodalFrontOnly(t *testing.T) {
	front := pngBytes(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		msgs, ok := body["messages"].([]any)
		if !ok || len(msgs) < 2 {
			t.Errorf("expected messages array with 2+ entries, got %#v", body["messages"])
			http.Error(w, "bad messages", http.StatusBadRequest)
			return
		}
		sys, _ := msgs[0].(map[string]any)
		if sys["role"] != "system" {
			t.Errorf("messages[0] role want system got %v", sys["role"])
		}
		if _, ok := sys["content"].(string); !ok {
			t.Errorf("messages[0] content want string")
		}
		user, _ := msgs[1].(map[string]any)
		if user["role"] != "user" {
			t.Errorf("messages[1] role want user got %v", user["role"])
		}
		parts, ok := user["content"].([]any)
		if !ok {
			t.Fatalf("user content want array, got %T", user["content"])
		}
		if n := countImageURLParts(parts); n != 1 {
			t.Fatalf("want 1 image_url part, got %d", n)
		}
		if !firstPartIsText(parts) {
			t.Fatal("expected first content part to be type text")
		}
		for _, p := range parts {
			m, ok := p.(map[string]any)
			if !ok {
				continue
			}
			if m["type"] != "image_url" {
				continue
			}
			iu, _ := m["image_url"].(map[string]any)
			url, _ := iu["url"].(string)
			if !strings.HasPrefix(url, "data:image/") {
				t.Fatalf("image url want data:image/ prefix, got %q", truncate(url, 48))
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"surface_score\":7,\"confidence\":0.6,\"evidence\":[\"test\"]}"}}]}`))
	}))
	t.Cleanup(srv.Close)

	c := NewClient(Config{
		BaseURL: srv.URL,
		Model:   "test-model",
	})
	out, err := c.AssessSurface(t.Context(), grading.AIAssistRequest{FrontImage: front})
	if err != nil {
		t.Fatal(err)
	}
	if out.SurfaceScore != 7 || out.Confidence != 0.6 {
		t.Fatalf("unexpected response: %+v", out)
	}
}

func TestAssessSurfaceRequestMultimodalFrontAndBack(t *testing.T) {
	front := pngBytes(t)
	back := pngBytes(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		msgs, _ := body["messages"].([]any)
		user, _ := msgs[1].(map[string]any)
		parts, _ := user["content"].([]any)
		if n := countImageURLParts(parts); n != 2 {
			t.Fatalf("want 2 image_url parts, got %d", n)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"surface_score\":8,\"confidence\":0.7,\"evidence\":[\"a\"]}"}}]}`))
	}))
	t.Cleanup(srv.Close)

	c := NewClient(Config{BaseURL: srv.URL, Model: "m"})
	_, err := c.AssessSurface(t.Context(), grading.AIAssistRequest{FrontImage: front, BackImage: back})
	if err != nil {
		t.Fatal(err)
	}
}

func countImageURLParts(parts []any) int {
	n := 0
	for _, p := range parts {
		m, ok := p.(map[string]any)
		if !ok {
			continue
		}
		if m["type"] == "image_url" {
			n++
		}
	}
	return n
}

func firstPartIsText(parts []any) bool {
	if len(parts) == 0 {
		return false
	}
	m, ok := parts[0].(map[string]any)
	return ok && m["type"] == "text"
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
