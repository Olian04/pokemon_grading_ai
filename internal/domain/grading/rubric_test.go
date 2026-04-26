package grading

import "testing"

func TestSellerConditionFromScore(t *testing.T) {
	tests := []struct {
		name  string
		score float64
		want  string
	}{
		{name: "mint", score: 9.7, want: "Mint"},
		{name: "nm", score: 8.8, want: "NM"},
		{name: "lp", score: 7.4, want: "LP"},
		{name: "mp", score: 6.2, want: "MP"},
		{name: "hp", score: 4.5, want: "HP"},
		{name: "damaged", score: 2.1, want: "Damaged"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sellerConditionFromScore(tt.score)
			if got != tt.want {
				t.Fatalf("sellerConditionFromScore(%f) = %q, want %q", tt.score, got, tt.want)
			}
		})
	}
}

func TestWeightedOverallScore(t *testing.T) {
	s := weightedOverallScore(map[string]float64{
		"centering": 9.0,
		"corners":   8.0,
		"edges":     7.0,
		"surface":   6.0,
	})
	if s <= 0 || s > 10 {
		t.Fatalf("weightedOverallScore returned out-of-range %f", s)
	}
	if s >= 9.0 || s <= 6.0 {
		t.Fatalf("weightedOverallScore expected weighted average between 6 and 9, got %f", s)
	}
}
