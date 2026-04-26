package openai

import "testing"

func TestParseAIAssistResponseJSON(t *testing.T) {
	out := parseAIAssistResponse(`{"surface_score":6.8,"confidence":0.77,"evidence":["print line"]}`)
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
