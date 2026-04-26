package grading

func weightedOverallScore(subscores map[string]float64) float64 {
	// Favor surface and centering because they correlate most with market spread.
	weights := map[string]float64{
		"centering": 0.30,
		"corners":   0.20,
		"edges":     0.20,
		"surface":   0.30,
	}
	var weighted float64
	var total float64
	for key, w := range weights {
		weighted += subscores[key] * w
		total += w
	}
	if total == 0 {
		return 1
	}
	score := weighted / total
	if score < 1 {
		return 1
	}
	if score > 10 {
		return 10
	}
	return score
}

func sellerConditionFromScore(score float64) string {
	switch {
	case score >= 9.5:
		return "Mint"
	case score >= 8.5:
		return "NM"
	case score >= 7.0:
		return "LP"
	case score >= 5.5:
		return "MP"
	case score >= 3.5:
		return "HP"
	default:
		return "Damaged"
	}
}
