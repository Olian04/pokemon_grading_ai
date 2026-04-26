package grading

import (
	"context"
	"errors"
	"math"
	"slices"

	"pokemon_ai/internal/domain/imageproc"
	"pokemon_ai/internal/integrations/market"
	"pokemon_ai/internal/integrations/pokemontcg"
)

type AIAdapter interface {
	AssessSurface(ctx context.Context, req AIAssistRequest) (AIAssistResponse, error)
}

type EventPublisher interface{}

type ImageAnalyzer interface {
	Analyze(path string) (imageproc.Result, error)
}

type Dependencies struct {
	AI        AIAdapter
	TCG       TCGAdapter
	Market    MarketAdapter
	Events    EventPublisher
	Analyzer  ImageAnalyzer
	PriceRule string
	ConfRule  string
	ScoreRule string
}

type Service struct {
	ai        AIAdapter
	tcg       TCGAdapter
	market    MarketAdapter
	analyzer  ImageAnalyzer
	priceRule string
	confRule  string
	scoreRule string
}

func NewService(deps Dependencies) *Service {
	return &Service{
		ai:        deps.AI,
		tcg:       deps.TCG,
		market:    deps.Market,
		analyzer:  deps.Analyzer,
		priceRule: deps.PriceRule,
		confRule:  deps.ConfRule,
		scoreRule: deps.ScoreRule,
	}
}

type TCGAdapter interface {
	SearchCards(ctx context.Context, query string) ([]pokemontcg.Card, error)
	GetCardPricing(ctx context.Context, id string) (pokemontcg.PriceSummary, error)
}

type MarketAdapter interface {
	BuildMarketResult(ctx context.Context, us pokemontcg.PriceSummary) market.Result
}

type GradeRequest struct {
	FrontImagePath string `json:"front_image_path"`
	BackImagePath  string `json:"back_image_path,omitempty"`
	CardNameHint   string `json:"card_name_hint,omitempty"`
	SetCodeHint    string `json:"set_code_hint,omitempty"`
	CardNumberHint string `json:"card_number_hint,omitempty"`
}

type GradeResponse struct {
	OverallProxy1To10 float64            `json:"overall_proxy_1_to_10"`
	SellerCondition   string             `json:"seller_condition"`
	Subscores         map[string]float64 `json:"subscores"`
	Confidence        float64            `json:"confidence"`
	Evidence          []string           `json:"evidence"`
	DeterministicOnly bool               `json:"deterministic_only"`
	AIUsed            bool               `json:"ai_used"`
	SkippedReason     string             `json:"skipped_reason,omitempty"`
	Market            market.Result      `json:"market"`
	Card              map[string]string  `json:"card"`
}

type AIAssistRequest struct {
	FrontImagePath string
	BackImagePath  string
}

type AIAssistResponse struct {
	SurfaceScore float64
	Evidence     []string
	Confidence   float64
}

func (s *Service) GradeCard(ctx context.Context, req GradeRequest) (GradeResponse, error) {
	if req.FrontImagePath == "" {
		return GradeResponse{}, errors.New("front_image_path is required")
	}
	if s.analyzer == nil {
		return GradeResponse{}, errors.New("image analyzer is not configured")
	}
	analysis, err := s.analyzer.Analyze(req.FrontImagePath)
	if err != nil {
		return GradeResponse{}, err
	}
	subscores := map[string]float64{
		"centering": analysis.CenteringScore,
		"corners":   analysis.CornersScore,
		"edges":     analysis.EdgesScore,
		"surface":   analysis.SurfaceScore,
	}
	overall := weightedOverallScore(subscores)
	deterministicOnly := true
	aiUsed := false
	skippedReason := ""
	confidence := analysis.Confidence
	evidence := slices.Clone(analysis.Evidence)
	marketResult, marketValueUSD := s.resolveMarket(ctx, req)

	eligibleByPrice, err := evaluateExpression(defaultRule(s.priceRule, ">= 20"), marketValueUSD)
	if err != nil {
		return GradeResponse{}, err
	}
	if !eligibleByPrice {
		skippedReason = "low_value"
	} else if s.ai != nil {
		confGate, err := evaluateExpression(defaultRule(s.confRule, "< 0.75"), analysis.Confidence)
		if err != nil {
			return GradeResponse{}, err
		}
		scoreGate, err := evaluateExpression(defaultRule(s.scoreRule, ">= 7.5"), overall)
		if err != nil {
			return GradeResponse{}, err
		}
		if confGate && scoreGate {
			aiResp, err := s.ai.AssessSurface(ctx, AIAssistRequest{
				FrontImagePath: req.FrontImagePath,
				BackImagePath:  req.BackImagePath,
			})
			if err == nil {
				subscores["surface"] = aiResp.SurfaceScore
				overall = weightedOverallScore(subscores)
				deterministicOnly = false
				aiUsed = true
				confidence = math.Round(((analysis.Confidence+aiResp.Confidence)/2)*100) / 100
				evidence = append(evidence, aiResp.Evidence...)
			}
		}
	}

	return GradeResponse{
		OverallProxy1To10: math.Round(overall*10) / 10,
		SellerCondition:   sellerConditionFromScore(overall),
		Subscores:         subscores,
		Confidence:        confidence,
		Evidence:          evidence,
		DeterministicOnly: deterministicOnly,
		AIUsed:            aiUsed,
		SkippedReason:     skippedReason,
		Market:            marketResult,
		Card: map[string]string{
			"name": req.CardNameHint,
			"set":  req.SetCodeHint,
		},
	}, nil
}

func (s *Service) resolveMarket(ctx context.Context, req GradeRequest) (market.Result, float64) {
	if s.tcg == nil || s.market == nil || req.CardNameHint == "" {
		return market.Result{
			EU: market.RegionStats{UnavailableReason: "cardmarket unavailable: missing pricing context"},
			US: market.RegionStats{UnavailableReason: "us pricing unavailable: missing card hint"},
		}, 0
	}
	cards, err := s.tcg.SearchCards(ctx, req.CardNameHint)
	if err != nil || len(cards) == 0 {
		return market.Result{
			EU: market.RegionStats{UnavailableReason: "cardmarket unavailable: missing pricing context"},
			US: market.RegionStats{UnavailableReason: "us pricing unavailable: search failed"},
		}, 0
	}
	price, err := s.tcg.GetCardPricing(ctx, cards[0].ID)
	if err != nil {
		return market.Result{
			EU: market.RegionStats{UnavailableReason: "cardmarket unavailable: missing pricing context"},
			US: market.RegionStats{UnavailableReason: "us pricing unavailable: pricing lookup failed"},
		}, 0
	}
	result := s.market.BuildMarketResult(ctx, price)
	var usd float64
	if result.US.CurrentMarketValue != nil {
		usd = *result.US.CurrentMarketValue
	}
	return result, usd
}

func defaultRule(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
