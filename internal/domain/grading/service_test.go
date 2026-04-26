package grading

import (
	"context"
	"testing"

	"pokemon_ai/internal/domain/imageproc"
)

type fakeAnalyzer struct {
	out imageproc.Result
	err error
}

func (f fakeAnalyzer) Analyze(_ string) (imageproc.Result, error) {
	return f.out, f.err
}

type fakeAI struct {
	out    AIAssistResponse
	err    error
	called bool
}

func (f *fakeAI) AssessSurface(_ context.Context, _ AIAssistRequest) (AIAssistResponse, error) {
	f.called = true
	return f.out, f.err
}

func TestGradeCardDeterministic(t *testing.T) {
	svc := NewService(Dependencies{
		Analyzer: fakeAnalyzer{
			out: imageproc.Result{
				CenteringScore: 8.5,
				CornersScore:   8.0,
				EdgesScore:     7.5,
				SurfaceScore:   7.0,
				Confidence:     0.85,
				Evidence:       []string{"synthetic metrics"},
			},
		},
	})

	resp, err := svc.GradeCard(context.Background(), GradeRequest{FrontImagePath: "front.png"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.SellerCondition == "" {
		t.Fatal("expected seller condition")
	}
	if resp.OverallProxy1To10 <= 0 {
		t.Fatal("expected overall score")
	}
	if len(resp.Evidence) == 0 {
		t.Fatal("expected evidence")
	}
}

func TestGradeCardUsesAIFallbackForLowConfidence(t *testing.T) {
	ai := &fakeAI{
		out: AIAssistResponse{
			SurfaceScore: 5.5,
			Confidence:   0.7,
			Evidence:     []string{"ai detected potential scratches"},
		},
	}
	svc := NewService(Dependencies{
		AI: ai,
		Analyzer: fakeAnalyzer{
			out: imageproc.Result{
				CenteringScore: 8.5,
				CornersScore:   8.0,
				EdgesScore:     7.5,
				SurfaceScore:   7.0,
				Confidence:     0.6,
				Evidence:       []string{"deterministic uncertain due glare"},
			},
		},
	})

	resp, err := svc.GradeCard(context.Background(), GradeRequest{
		FrontImagePath: "front.png",
		BackImagePath:  "back.png",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ai.called {
		t.Fatal("expected AI fallback call")
	}
	if resp.DeterministicOnly {
		t.Fatal("expected deterministic_only=false when AI fallback is used")
	}
	if resp.Subscores["surface"] != 5.5 {
		t.Fatalf("expected merged AI surface score 5.5, got %f", resp.Subscores["surface"])
	}
}
