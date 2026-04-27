package openai

import (
	"encoding/json"
	"fmt"
	"strings"

	"pokemon_ai/internal/domain/grading"
)

const surfacePromptSystem = `You are a Pokemon card condition assistant.
Respond in JSON only with keys: surface_score, confidence, evidence.
surface_score range is 1-10. confidence range is 0-1.
evidence is a short array of strings.`

func buildSurfacePrompt(req grading.AIAssistRequest) string {
	parts := []string{
		"Analyze card surface defects from provided file paths metadata.",
		"Front image path: " + req.FrontImagePath,
	}
	if req.BackImagePath != "" {
		parts = append(parts, "Back image path: "+req.BackImagePath)
	}
	parts = append(parts, "If uncertain, use lower confidence.")
	return strings.Join(parts, "\n")
}

func parseAIAssistResponse(raw string) (grading.AIAssistResponse, error) {
	var parsed struct {
		SurfaceScore float64  `json:"surface_score"`
		Confidence   float64  `json:"confidence"`
		Evidence     []string `json:"evidence"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return grading.AIAssistResponse{}, fmt.Errorf("%w: %w", ErrInvalidAIAssistJSON, err)
	}
	if parsed.SurfaceScore <= 0 || parsed.SurfaceScore > 10 {
		return grading.AIAssistResponse{}, fmt.Errorf("%w: got %g", ErrInvalidSurfaceScore, parsed.SurfaceScore)
	}
	if parsed.Confidence < 0 || parsed.Confidence > 1 {
		return grading.AIAssistResponse{}, fmt.Errorf("%w: got %g", ErrInvalidConfidenceScore, parsed.Confidence)
	}
	if len(parsed.Evidence) == 0 {
		return grading.AIAssistResponse{}, ErrNoEvidenceProvided
	}
	return grading.AIAssistResponse{
		SurfaceScore: parsed.SurfaceScore,
		Confidence:   parsed.Confidence,
		Evidence:     parsed.Evidence,
	}, nil
}
