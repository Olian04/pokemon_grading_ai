package market

import (
	"context"
	"math"

	"pokemon_ai/internal/integrations/pokemontcg"
)

type RegionStats struct {
	CurrentMarketValue *float64 `json:"current_market_value,omitempty"`
	TradeVolume        *int     `json:"trade_volume,omitempty"`
	Variance           *float64 `json:"variance,omitempty"`
	StdDeviation       *float64 `json:"std_deviation,omitempty"`
	UnavailableReason  string   `json:"unavailable_reason,omitempty"`
}

type Result struct {
	US RegionStats `json:"us"`
	EU RegionStats `json:"eu"`
}

type Config struct {
	CardmarketBaseURL string
	CardmarketAPIKey  string
}

type Service struct {
	cardmarketBaseURL string
	cardmarketAPIKey  string
}

func NewService(cfg Config) *Service {
	return &Service{
		cardmarketBaseURL: cfg.CardmarketBaseURL,
		cardmarketAPIKey:  cfg.CardmarketAPIKey,
	}
}

func (s *Service) BuildMarketResult(_ context.Context, us pokemontcg.PriceSummary) Result {
	var usValue *float64
	if us.Holofoil != nil {
		usValue = us.Holofoil
	} else if us.Normal != nil {
		usValue = us.Normal
	} else {
		usValue = us.Reverse
	}
	volume := 0
	if usValue != nil {
		volume = 1
	}
	var variance, stddev *float64
	if us.Normal != nil || us.Holofoil != nil || us.Reverse != nil {
		values := make([]float64, 0, 3)
		for _, v := range []*float64{us.Normal, us.Holofoil, us.Reverse} {
			if v != nil {
				values = append(values, *v)
			}
		}
		if len(values) > 0 {
			v, sdev := calcVariance(values)
			variance = &v
			stddev = &sdev
		}
	}

	eu := RegionStats{
		UnavailableReason: "cardmarket data unavailable: API integration requires credentials and product mapping",
	}
	if s.cardmarketAPIKey == "" || s.cardmarketBaseURL == "" {
		eu.UnavailableReason = "cardmarket unavailable: missing cardmarket API configuration"
	}

	var usVolumePtr *int
	if volume > 0 {
		usVolumePtr = &volume
	}
	return Result{
		US: RegionStats{
			CurrentMarketValue: usValue,
			TradeVolume:        usVolumePtr,
			Variance:           variance,
			StdDeviation:       stddev,
		},
		EU: eu,
	}
}

func calcVariance(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}
	var mean float64
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))
	var sq float64
	for _, v := range values {
		d := v - mean
		sq += d * d
	}
	variance := sq / float64(len(values))
	return variance, math.Sqrt(variance)
}
