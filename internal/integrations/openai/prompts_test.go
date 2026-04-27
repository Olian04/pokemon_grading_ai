package openai

import (
	"errors"
	"strings"
	"testing"

	"pokemon_ai/internal/domain/grading"
)

func TestParseAIAssistResponseJSON(t *testing.T) {
	out, err := parseAIAssistResponse(`{"surface_score":6.8,"confidence":0.77,"evidence":["print line"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if out.SurfaceScore != 6.8 {
		t.Fatalf("unexpected surface score: %f", out.SurfaceScore)
	}
	if out.Confidence != 0.77 {
		t.Fatalf("unexpected confidence: %f", out.Confidence)
	}
	if len(out.Evidence) != 1 {
		t.Fatalf("unexpected evidence length: %d", len(out.Evidence))
	}
}

func TestParseAIAssistResponseInvalidSurfaceScore(t *testing.T) {
	for _, raw := range []string{
		`{"surface_score":0,"confidence":0.5,"evidence":[]}`,
		`{"surface_score":-1,"confidence":0.5,"evidence":[]}`,
		`{"surface_score":11,"confidence":0.5,"evidence":[]}`,
		`{"confidence":0.5,"evidence":[]}`,
	} {
		_, err := parseAIAssistResponse(raw)
		if !errors.Is(err, ErrInvalidSurfaceScore) {
			t.Fatalf("expected ErrInvalidSurfaceScore for %q, got %v", raw, err)
		}
	}
}

func TestParseAIAssistResponseInvalidJSON(t *testing.T) {
	_, err := parseAIAssistResponse(`not json`)
	if !errors.Is(err, ErrInvalidAIAssistJSON) {
		t.Fatalf("expected ErrInvalidAIAssistJSON, got %v", err)
	}
}

func TestParseAIAssistResponseEmptyInput(t *testing.T) {
	_, err := parseAIAssistResponse("")
	if !errors.Is(err, ErrInvalidAIAssistJSON) {
		t.Fatalf("expected ErrInvalidAIAssistJSON, got %v", err)
	}
}

func TestParseAIAssistResponseBoundaryValid(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want grading.AIAssistResponse
	}{
		{
			name: "min surface and confidence",
			raw:  `{"surface_score":1,"confidence":0,"evidence":["edge wear"]}`,
			want: grading.AIAssistResponse{SurfaceScore: 1, Confidence: 0, Evidence: []string{"edge wear"}},
		},
		{
			name: "max surface and confidence",
			raw:  `{"surface_score":10,"confidence":1,"evidence":["clean"]}`,
			want: grading.AIAssistResponse{SurfaceScore: 10, Confidence: 1, Evidence: []string{"clean"}},
		},
		{
			name: "multiple evidence strings",
			raw:  `{"surface_score":7,"confidence":0.5,"evidence":["a","b"]}`,
			want: grading.AIAssistResponse{SurfaceScore: 7, Confidence: 0.5, Evidence: []string{"a", "b"}},
		},
		{
			name: "extra json keys ignored",
			raw:  `{"surface_score":5,"confidence":0.5,"evidence":["ok"],"note":"ignored"}`,
			want: grading.AIAssistResponse{SurfaceScore: 5, Confidence: 0.5, Evidence: []string{"ok"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseAIAssistResponse(tc.raw)
			if err != nil {
				t.Fatal(err)
			}
			if got.SurfaceScore != tc.want.SurfaceScore || got.Confidence != tc.want.Confidence {
				t.Fatalf("got score=%g conf=%g want score=%g conf=%g",
					got.SurfaceScore, got.Confidence, tc.want.SurfaceScore, tc.want.Confidence)
			}
			if len(got.Evidence) != len(tc.want.Evidence) {
				t.Fatalf("evidence len got %d want %d", len(got.Evidence), len(tc.want.Evidence))
			}
			for i := range tc.want.Evidence {
				if got.Evidence[i] != tc.want.Evidence[i] {
					t.Fatalf("evidence[%d] got %q want %q", i, got.Evidence[i], tc.want.Evidence[i])
				}
			}
		})
	}
}

func TestParseAIAssistResponseInvalidConfidence(t *testing.T) {
	for _, raw := range []string{
		`{"surface_score":5,"confidence":-0.01,"evidence":["x"]}`,
		`{"surface_score":5,"confidence":1.01,"evidence":["x"]}`,
		`{"surface_score":5,"confidence":2,"evidence":["x"]}`,
	} {
		_, err := parseAIAssistResponse(raw)
		if !errors.Is(err, ErrInvalidConfidenceScore) {
			t.Fatalf("expected ErrInvalidConfidenceScore for %q, got %v", raw, err)
		}
	}
}

func TestParseAIAssistResponseNoEvidence(t *testing.T) {
	for _, raw := range []string{
		`{"surface_score":5,"confidence":0.5,"evidence":[]}`,
		`{"surface_score":5,"confidence":0.5}`,
	} {
		_, err := parseAIAssistResponse(raw)
		if !errors.Is(err, ErrNoEvidenceProvided) {
			t.Fatalf("expected ErrNoEvidenceProvided for %q, got %v", raw, err)
		}
	}
}

func TestParseAIAssistResponseOmittedConfidenceIsZero(t *testing.T) {
	// Omitted confidence unmarshals as 0, which is valid per 0..1.
	out, err := parseAIAssistResponse(`{"surface_score":5,"evidence":["scratch"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if out.Confidence != 0 {
		t.Fatalf("confidence: got %g want 0", out.Confidence)
	}
}

func TestBuildSurfacePromptFrontOnly(t *testing.T) {
	got := buildSurfacePrompt(grading.AIAssistRequest{FrontImage: []byte{1}})
	if strings.Contains(got, "second image") {
		t.Fatalf("did not expect second image wording: %q", got)
	}
	for _, want := range []string{
		"The first image is always the card front",
		"Only the front was provided",
		"If uncertain, use lower confidence.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("prompt missing %q:\n%s", want, got)
		}
	}
}

func TestBuildSurfacePromptWithBack(t *testing.T) {
	got := buildSurfacePrompt(grading.AIAssistRequest{
		FrontImage: []byte{1},
		BackImage:  []byte{2},
	})
	for _, want := range []string{
		"The first image is always the card front",
		"The second image is the card back.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("prompt missing %q:\n%s", want, got)
		}
	}
}

func TestSurfacePromptSystemNonEmpty(t *testing.T) {
	if len(surfacePromptSystem) < 20 {
		t.Fatalf("unexpectedly short system prompt: %q", surfacePromptSystem)
	}
	for _, token := range []string{"JSON", "surface_score", "confidence", "evidence"} {
		if !strings.Contains(surfacePromptSystem, token) {
			t.Fatalf("system prompt should mention %q", token)
		}
	}
}
