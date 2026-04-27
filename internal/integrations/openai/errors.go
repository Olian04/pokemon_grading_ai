package openai

import "errors"

var (
	// ErrEmptyBaseURL is returned when the client is configured with no base URL.
	ErrEmptyBaseURL = errors.New("openai: base url is empty")

	// ErrHTTPError indicates a non-success HTTP status from the completions API.
	// Wrap with fmt.Errorf("%w: status %d", ErrHTTPError, code) so callers can use errors.Is.
	ErrHTTPError = errors.New("openai: request failed")

	// ErrNoCompletionChoices is returned when the API response contains no choices.
	ErrNoCompletionChoices = errors.New("openai: response had no choices")

	// ErrInvalidAIAssistJSON is returned when the model content is not valid JSON for the assist schema.
	// Wrap the underlying json.Unmarshal error with fmt.Errorf("%w: %w", ErrInvalidAIAssistJSON, err).
	ErrInvalidAIAssistJSON = errors.New("openai: invalid ai assist response json")

	// ErrInvalidSurfaceScore is returned when surface_score is missing or outside 1..10.
	// Wrap with fmt.Errorf("%w: got %g", ErrInvalidSurfaceScore, v) when a concrete value helps operators.
	ErrInvalidSurfaceScore = errors.New("openai: invalid surface_score (expected 1..10)")

	// ErrInvalidConfidenceScore is returned when confidence is missing or outside 0..1.
	// Wrap with fmt.Errorf("%w: got %g", ErrInvalidConfidenceScore, v) when a concrete value helps operators.
	ErrInvalidConfidenceScore = errors.New("openai: invalid confidence (expected 0..1)")

	// ErrNoEvidenceProvided is returned when evidence is missing or empty.
	// Wrap with fmt.Errorf("%w: got empty evidence", ErrNoEvidenceProvided) when a concrete value helps operators.
	ErrNoEvidenceProvided = errors.New("openai: no evidence provided")
)
