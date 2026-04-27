package imageproc

import "errors"

var (
	// ErrNoCardQuad is returned when strict card normalization cannot find a plausible card quadrilateral.
	ErrNoCardQuad = errors.New("imageproc: no card quadrilateral detected")

	// ErrDegenerateHomography is returned when the perspective transform is numerically unusable.
	ErrDegenerateHomography = errors.New("imageproc: degenerate homography")

	// ErrInvalidDebugOutputDir is returned when debug_normalize is enabled but output_dir is empty or invalid.
	ErrInvalidDebugOutputDir = errors.New("imageproc: debug_normalize.output_dir required when enabled")
)
