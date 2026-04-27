package imageproc

import "image"

// Tier B (future): detect the inner Pokémon print border (yellow/white frame) on the dewarped
// card image and score true centering as symmetric margins between print box and card edge.
// See docs/AGENT_CONTEXT.md. Current pipeline uses Tier A (post-dewarp luminance heuristics).

// EstimatePrintBorderCentering is reserved for Tier B: measure print-frame margins on a rectified card.
// Not implemented; callers should ignore the result when ok is false.
func EstimatePrintBorderCentering(img image.Image) (centering01 float64, ok bool) {
	_ = img
	return 0, false
}
