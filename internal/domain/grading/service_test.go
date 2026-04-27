package grading

import (
	"context"
	"testing"

	"pokemon_ai/internal/domain/imageproc"
	"pokemon_ai/internal/integrations/market"
	"pokemon_ai/internal/integrations/pokemontcg"
)

type fakeAnalyzer struct {
	out imageproc.Result
	err error
}

func (f fakeAnalyzer) Analyze(_ []byte) (imageproc.Result, error) {
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

type fakeTCG struct{}

func (fakeTCG) SearchCards(_ context.Context, _ string) ([]pokemontcg.Card, error) {
	return []pokemontcg.Card{{ID: "base1-4", Name: "Charizard"}}, nil
}

func (fakeTCG) GetCardPricing(_ context.Context, _ string) (pokemontcg.PriceSummary, error) {
	v := 100.0
	return pokemontcg.PriceSummary{ID: "base1-4", Holofoil: &v}, nil
}

type recordingTCG struct {
	lastPricingID string
}

func (recordingTCG) SearchCards(_ context.Context, _ string) ([]pokemontcg.Card, error) {
	return []pokemontcg.Card{
		{ID: "base1-4", Name: "Charizard"},
		{ID: "sv1-25", Name: "Pikachu"},
	}, nil
}

func (r *recordingTCG) GetCardPricing(_ context.Context, id string) (pokemontcg.PriceSummary, error) {
	r.lastPricingID = id
	v := 10.0
	return pokemontcg.PriceSummary{ID: id, Normal: &v}, nil
}

type fakeMarket struct{}

func (fakeMarket) BuildMarketResult(_ context.Context, in market.BuildInput) market.Result {
	return market.Result{
		US: market.RegionStats{
			CurrentMarketValue: in.US.Holofoil,
		},
		EU: market.RegionStats{
			UnavailableReason: "cardmarket unavailable for test",
		},
	}
}

func TestGradeCardDeterministic(t *testing.T) {
	svc := NewService(Dependencies{
		TCG:    fakeTCG{},
		Market: fakeMarket{},
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

	resp, err := svc.GradeCard(context.Background(), GradeRequest{FrontImage: []byte("front")})
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
		AI:        ai,
		TCG:       fakeTCG{},
		Market:    fakeMarket{},
		ConfRule:  "< 0.75",
		ScoreRule: ">= 7.5",
		PriceRule: ">= 50",
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
		FrontImage:   []byte("front"),
		BackImage:    []byte("back"),
		CardNameHint: "Charizard",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ai.called {
		t.Fatal("expected AI fallback call")
	}
	if resp.DeterministicOnly || !resp.AIUsed {
		t.Fatal("expected deterministic_only=false when AI fallback is used")
	}
	if resp.Subscores["surface"] != 5.5 {
		t.Fatalf("expected merged AI surface score 5.5, got %f", resp.Subscores["surface"])
	}
}

func TestResolveMarketPicksCardMatchingSetHint(t *testing.T) {
	tcg := &recordingTCG{}
	svc := NewService(Dependencies{
		TCG:      tcg,
		Market:   fakeMarket{},
		Analyzer: fakeAnalyzer{out: imageproc.Result{CenteringScore: 8, CornersScore: 8, EdgesScore: 8, SurfaceScore: 8, Confidence: 0.9}},
	})
	_, usd := svc.resolveMarket(context.Background(), GradeRequest{
		CardNameHint:   "Pikachu",
		SetCodeHint:    "sv1",
		CardNumberHint: "25",
	})
	if tcg.lastPricingID != "sv1-25" {
		t.Fatalf("expected pricing for sv1-25, got %q (usd=%v)", tcg.lastPricingID, usd)
	}
}

func TestGradeCardSkipsAIWhenLowValue(t *testing.T) {
	ai := &fakeAI{
		out: AIAssistResponse{SurfaceScore: 4.0, Confidence: 0.9},
	}
	svc := NewService(Dependencies{
		AI:        ai,
		TCG:       fakeTCG{},
		Market:    fakeMarket{},
		PriceRule: ">= 500",
		ConfRule:  "< 0.75",
		ScoreRule: ">= 7.5",
		Analyzer: fakeAnalyzer{
			out: imageproc.Result{
				CenteringScore: 8.0,
				CornersScore:   8.0,
				EdgesScore:     8.0,
				SurfaceScore:   8.0,
				Confidence:     0.1,
				Evidence:       []string{"low confidence"},
			},
		},
	})

	resp, err := svc.GradeCard(context.Background(), GradeRequest{FrontImage: []byte("front"), CardNameHint: "Charizard"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ai.called {
		t.Fatal("expected AI not to be called for low-value skip")
	}
	if resp.SkippedReason != "low_value" {
		t.Fatalf("expected skipped_reason low_value, got %q", resp.SkippedReason)
	}
}
