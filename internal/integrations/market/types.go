package market

import (
	"context"
	"fmt"
	"math"
	"time"

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

// BuildInput carries US TCGPlayer-derived prices plus the TCG card identity used for Cardmarket mapping.
type BuildInput struct {
	US          pokemontcg.PriceSummary
	Card        pokemontcg.Card
	SetCodeHint string
	NumberHint  string
}

// CardmarketOAuthConfig holds MKM API 2.0 OAuth 1.0a credentials (app + access token pairs).
type CardmarketOAuthConfig struct {
	BaseURL      string
	AppToken     string
	AppSecret    string
	AccessToken  string
	AccessSecret string
	HTTPTimeout  time.Duration
}

type Config struct {
	Cardmarket CardmarketOAuthConfig
	// IDGame is the Cardmarket game id (Pokemon TCG is typically 3; confirm in MKM account/docs).
	IDGame int
	// SinglesCacheTTL controls in-memory caching of expansion singles lists.
	SinglesCacheTTL time.Duration
	// TcgSetToExpansion maps Pokemon TCG API set codes (e.g. "base1") to Cardmarket idExpansion.
	// YAML keys under market.tcg_set_to_expansion merge over embedded defaults.
	TcgSetToExpansion map[string]int
}

type Service struct {
	cm           *CardmarketClient
	expansionMap map[string]int
	singlesCache *singlesCache
}

func NewService(cfg Config) (*Service, error) {
	defaultMap, err := loadDefaultExpansionMap()
	if err != nil {
		return nil, err
	}
	expMap := mergeExpansionMaps(defaultMap, cfg.TcgSetToExpansion)

	ttl := cfg.SinglesCacheTTL
	if ttl <= 0 {
		ttl = time.Hour
	}
	svc := &Service{
		expansionMap: expMap,
		singlesCache: newSinglesCache(ttl),
	}
	_ = cfg.IDGame // reserved for future MKM list-by-game endpoints; YAML default is set in bootstrap.
	if cfg.Cardmarket.AppToken != "" && cfg.Cardmarket.AppSecret != "" &&
		cfg.Cardmarket.AccessToken != "" && cfg.Cardmarket.AccessSecret != "" {
		cm, err := NewCardmarketClient(cfg.Cardmarket)
		if err != nil {
			return nil, err
		}
		svc.cm = cm
	}
	return svc, nil
}

// BuildMarketResult aggregates US prices from TCGPlayer data and, when configured, EU trend from Cardmarket.
func (s *Service) BuildMarketResult(ctx context.Context, in BuildInput) Result {
	us := aggregateUSFromSummary(in.US)

	eu := RegionStats{}
	if s.cm == nil {
		eu.UnavailableReason = "cardmarket unavailable: missing oauth configuration"
		return Result{US: us, EU: eu}
	}

	setCode, _ := tcgSetAndLocal(in.Card.ID)
	if setCode == "" {
		eu.UnavailableReason = "cardmarket unavailable: invalid tcg card id"
		return Result{US: us, EU: eu}
	}
	idExpansion, ok := s.expansionMap[setCode]
	if !ok || idExpansion <= 0 {
		eu.UnavailableReason = "cardmarket unavailable: unknown tcg set mapping"
		return Result{US: us, EU: eu}
	}

	rows, hit := s.singlesCache.get(idExpansion)
	if !hit {
		var err error
		rows, err = s.cm.FetchExpansionSingles(ctx, idExpansion)
		if err != nil {
			eu.UnavailableReason = fmt.Sprintf("cardmarket unavailable: %v", err)
			return Result{US: us, EU: eu}
		}
		s.singlesCache.set(idExpansion, rows)
	}

	row, ok := pickSingleRow(rows, in.Card, in.SetCodeHint, in.NumberHint)
	if !ok {
		eu.UnavailableReason = "cardmarket unavailable: no matching product in expansion"
		return Result{US: us, EU: eu}
	}

	var eur float64
	if row.Price != nil && row.Price.Trend != nil {
		eur = *row.Price.Trend
	} else {
		v, err := s.cm.FetchProductTrendEUR(ctx, row.IDProduct)
		if err != nil {
			eu.UnavailableReason = fmt.Sprintf("cardmarket unavailable: %v", err)
			return Result{US: us, EU: eu}
		}
		eur = v
	}
	// EU value: Cardmarket "trend" in EUR (account currency); see product/singles JSON in MKM docs.
	eu.CurrentMarketValue = &eur
	return Result{US: us, EU: eu}
}

func aggregateUSFromSummary(us pokemontcg.PriceSummary) RegionStats {
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
	var usVolumePtr *int
	if volume > 0 {
		usVolumePtr = &volume
	}
	return RegionStats{
		CurrentMarketValue: usValue,
		TradeVolume:        usVolumePtr,
		Variance:           variance,
		StdDeviation:       stddev,
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
