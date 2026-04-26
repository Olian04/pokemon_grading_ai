package grading

import "testing"

func TestEvaluateExpression(t *testing.T) {
	tests := []struct {
		name   string
		expr   string
		left   float64
		want   bool
		hasErr bool
	}{
		{name: "less than", expr: "< 0.75", left: 0.6, want: true},
		{name: "greater equal", expr: ">= 8.5", left: 8.5, want: true},
		{name: "bad operator", expr: "== 1", left: 1, hasErr: true},
		{name: "bad syntax", expr: "<0.5", left: 0.1, hasErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evaluateExpression(tt.expr, tt.left)
			if tt.hasErr {
				if err == nil {
					t.Fatalf("expected error for %q", tt.expr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("evaluateExpression(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}
