package grading

import (
	"context"
	"errors"
	"math"
	"slices"

	"pokemon_ai/internal/domain/imageproc"
)

type AIAdapter interface {
	AssessSurface(ctx context.Context, req AIAssistRequest) (AIAssistResponse, error)
}

type EventPublisher interface{}

type ImageAnalyzer interface {
	Analyze(path string) (imageproc.Result, error)
}

type Dependencies struct {
	AI       AIAdapter
	TCG      any
	Events   EventPublisher
	Analyzer ImageAnalyzer
}

type Service struct {
	ai       AIAdapter
	analyzer ImageAnalyzer
}

func NewService(deps Dependencies) *Service {
	return &Service{
		ai:       deps.AI,
		analyzer: deps.Analyzer,
	}
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
	confidence := analysis.Confidence
	evidence := slices.Clone(analysis.Evidence)

	if analysis.Confidence < 0.75 && s.ai != nil {
		aiResp, err := s.ai.AssessSurface(ctx, AIAssistRequest{
			FrontImagePath: req.FrontImagePath,
			BackImagePath:  req.BackImagePath,
		})
		if err == nil {
			subscores["surface"] = aiResp.SurfaceScore
			overall = weightedOverallScore(subscores)
			deterministicOnly = false
			confidence = math.Round(((analysis.Confidence+aiResp.Confidence)/2)*100) / 100
			evidence = append(evidence, aiResp.Evidence...)
		}
	}

	return GradeResponse{
		OverallProxy1To10: math.Round(overall*10) / 10,
		SellerCondition:   sellerConditionFromScore(overall),
		Subscores:         subscores,
		Confidence:        confidence,
		Evidence:          evidence,
		DeterministicOnly: deterministicOnly,
		Card: map[string]string{
			"name": req.CardNameHint,
			"set":  req.SetCodeHint,
		},
	}, nil
}
