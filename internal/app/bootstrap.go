package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"pokemon_ai/internal/domain/grading"
	"pokemon_ai/internal/domain/imageproc"
	"pokemon_ai/internal/integrations/market"
	"pokemon_ai/internal/integrations/openai"
	"pokemon_ai/internal/integrations/pokemontcg"
	"pokemon_ai/internal/observability/metrics"
	httptransport "pokemon_ai/internal/transport/http"
	"pokemon_ai/internal/transport/http/handlers"
	mcptransport "pokemon_ai/internal/transport/mcp"

	"gopkg.in/yaml.v3"
)

type Config struct {
	HTTPAddr                        string
	ReadTimeout                     time.Duration
	WriteTimeout                    time.Duration
	LogLevel                        slog.Level
	HTTPAccessLogEnabled            bool
	HTTPSlowRequestThreshold        time.Duration
	OpenAIBaseURL                   string
	OpenAIAPIKey                    string
	OpenAIModel                     string
	AIPriceRule                     string
	AIConfidenceRule                string
	AIScoreRule                     string
	MarketCardmarketBaseURL         string
	MarketCardmarketAppToken        string
	MarketCardmarketAppSecret       string
	MarketCardmarketAccessToken     string
	MarketCardmarketAccessSecret    string
	MarketCardmarketIDGame          int
	MarketCardmarketHTTPTimeout     time.Duration
	MarketCardmarketSinglesCacheTTL time.Duration
	MarketTcgSetToExpansion         map[string]int
	PokemonTCGBaseURL               string
	PokemonTCGAPIKey                string
	FallbackRequestsPerMin          int
	EnableMCP                       bool
	Imageproc                       imageproc.Config
}

const configPathEnv = "APP_CONFIG_FILE"

func Bootstrap(ctx context.Context) (*http.Server, func(), error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, nil, err
	}
	return BootstrapWithConfig(ctx, cfg)
}

func BootstrapWithConfig(ctx context.Context, cfg Config) (*http.Server, func(), error) {
	tcgClient := pokemontcg.NewClient(pokemontcg.Config{
		BaseURL:                cfg.PokemonTCGBaseURL,
		APIKey:                 cfg.PokemonTCGAPIKey,
		FallbackRequestsPerMin: cfg.FallbackRequestsPerMin,
	})
	aiClient := openai.NewClient(openai.Config{
		BaseURL: cfg.OpenAIBaseURL,
		APIKey:  cfg.OpenAIAPIKey,
		Model:   cfg.OpenAIModel,
		Timeout: 30 * time.Second,
	})

	mkt, err := market.NewService(market.Config{
		Cardmarket: market.CardmarketOAuthConfig{
			BaseURL:      cfg.MarketCardmarketBaseURL,
			AppToken:     cfg.MarketCardmarketAppToken,
			AppSecret:    cfg.MarketCardmarketAppSecret,
			AccessToken:  cfg.MarketCardmarketAccessToken,
			AccessSecret: cfg.MarketCardmarketAccessSecret,
			HTTPTimeout:  cfg.MarketCardmarketHTTPTimeout,
		},
		IDGame:            cfg.MarketCardmarketIDGame,
		SinglesCacheTTL:   cfg.MarketCardmarketSinglesCacheTTL,
		TcgSetToExpansion: cfg.MarketTcgSetToExpansion,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("market service: %w", err)
	}

	service := grading.NewService(grading.Dependencies{
		AI:        aiClient,
		TCG:       tcgClient,
		Analyzer:  imageproc.NewAnalyzer(cfg.Imageproc),
		Market:    mkt,
		PriceRule: cfg.AIPriceRule,
		ConfRule:  cfg.AIConfidenceRule,
		ScoreRule: cfg.AIScoreRule,
	})

	h := handlers.New(handlers.Dependencies{
		Grading: service,
		TCG:     tcgClient,
	})

	registry := metrics.NewRegistry()
	router := httptransport.NewRouter(httptransport.RouterConfig{
		Handlers:             h,
		EnableMCP:            cfg.EnableMCP,
		MCPServer:            mcptransport.NewServer(service),
		AccessLogEnabled:     cfg.HTTPAccessLogEnabled,
		SlowRequestThreshold: cfg.HTTPSlowRequestThreshold,
		Metrics:              registry,
	})
	server := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	cleanup := func() {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}

	return server, cleanup, nil
}

func LoadConfig() (Config, error) {
	path, err := resolveConfigPath()
	if err != nil {
		return Config{}, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config file %q: %w", path, err)
	}
	var y yamlConfig
	if err := yaml.Unmarshal(raw, &y); err != nil {
		return Config{}, fmt.Errorf("parse config file %q: %w", path, err)
	}

	cfg := Config{
		HTTPAddr:                        firstNonEmpty(y.HTTP.Addr, ":8080"),
		ReadTimeout:                     parseDurationOrDefault(y.HTTP.ReadTimeout, 15*time.Second),
		WriteTimeout:                    parseDurationOrDefault(y.HTTP.WriteTimeout, 60*time.Second),
		LogLevel:                        parseLevel(y.Logging.Level, slog.LevelInfo),
		HTTPAccessLogEnabled:            defaultBool(y.Logging.HTTPAccessLogEnabled, true),
		HTTPSlowRequestThreshold:        parseDurationOrDefault(y.Logging.HTTPSlowRequestThreshold, 500*time.Millisecond),
		OpenAIBaseURL:                   firstNonEmpty(y.OpenAI.BaseURL, "http://localhost:11434/v1"),
		OpenAIAPIKey:                    y.OpenAI.APIKey,
		OpenAIModel:                     firstNonEmpty(y.OpenAI.Model, "qwen2.5:7b"),
		AIPriceRule:                     firstNonEmpty(y.AI.PriceRule, ">= 20"),
		AIConfidenceRule:                firstNonEmpty(y.AI.ConfidenceRule, "< 0.75"),
		AIScoreRule:                     firstNonEmpty(y.AI.ScoreRule, ">= 7.5"),
		MarketCardmarketBaseURL:         firstNonEmpty(y.Market.CardmarketBaseURL, "https://apiv2.cardmarket.com/ws/v2.0"),
		MarketCardmarketAppToken:        y.Market.CardmarketAppToken,
		MarketCardmarketAppSecret:       y.Market.CardmarketAppSecret,
		MarketCardmarketAccessToken:     y.Market.CardmarketAccessToken,
		MarketCardmarketAccessSecret:    y.Market.CardmarketAccessSecret,
		MarketCardmarketIDGame:          defaultInt(y.Market.CardmarketIDGame, 3),
		MarketCardmarketHTTPTimeout:     parseDurationOrDefault(y.Market.CardmarketHTTPTimeout, 20*time.Second),
		MarketCardmarketSinglesCacheTTL: parseDurationOrDefault(y.Market.CardmarketSinglesCacheTTL, time.Hour),
		MarketTcgSetToExpansion:         y.Market.TcgSetToExpansion,
		PokemonTCGBaseURL:               firstNonEmpty(y.PokemonTCG.BaseURL, "https://api.pokemontcg.io/v2"),
		PokemonTCGAPIKey:                y.PokemonTCG.APIKey,
		FallbackRequestsPerMin:          defaultInt(y.PokemonTCG.FallbackRequestsPerMin, 15),
		EnableMCP:                       defaultBool(y.MCP.Enable, false),
	}
	ip, errIP := buildImageprocConfig(y)
	if errIP != nil {
		return Config{}, errIP
	}
	cfg.Imageproc = ip
	if err := grading.ValidateExpression(cfg.AIPriceRule); err != nil {
		return Config{}, fmt.Errorf("invalid ai.price_rule: %w", err)
	}
	if err := grading.ValidateExpression(cfg.AIConfidenceRule); err != nil {
		return Config{}, fmt.Errorf("invalid ai.confidence_rule: %w", err)
	}
	if err := grading.ValidateExpression(cfg.AIScoreRule); err != nil {
		return Config{}, fmt.Errorf("invalid ai.score_rule: %w", err)
	}
	return cfg, nil
}

var configFilePaths = []string{
	"pokemon-ai.yaml",
	".pokemon-ai.yaml",
	filepath.Join(".pokemon-ai", "config.yaml"),
}

func resolveConfigPath() (string, error) {
	if explicit := strings.TrimSpace(os.Getenv(configPathEnv)); explicit != "" {
		return explicit, nil
	}
	candidates := make([]string, 0, 9)
	if wd, err := os.Getwd(); err == nil {
		for _, p := range configFilePaths {
			candidates = append(candidates, filepath.Join(wd, p))
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		for _, p := range configFilePaths {
			candidates = append(candidates, filepath.Join(home, p))
		}
	}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		for _, p := range configFilePaths {
			candidates = append(candidates, filepath.Join(exeDir, p))
		}
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", errors.New("config file not found; set APP_CONFIG_FILE or provide config.yaml in working dir/home/executable location")
}

func parseLevel(raw string, fallback slog.Level) slog.Level {
	switch raw {
	case "debug", "DEBUG":
		return slog.LevelDebug
	case "info", "INFO", "":
		return slog.LevelInfo
	case "warn", "WARN", "warning", "WARNING":
		return slog.LevelWarn
	case "error", "ERROR":
		return slog.LevelError
	default:
		return fallback
	}
}

func parseDurationOrDefault(raw string, fallback time.Duration) time.Duration {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return v
}

func firstNonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func defaultInt(v, fallback int) int {
	if v <= 0 {
		return fallback
	}
	return v
}

func defaultBool(v *bool, fallback bool) bool {
	if v == nil {
		return fallback
	}
	return *v
}

type yamlConfig struct {
	HTTP struct {
		Addr         string `yaml:"addr"`
		ReadTimeout  string `yaml:"read_timeout"`
		WriteTimeout string `yaml:"write_timeout"`
	} `yaml:"http"`
	Logging struct {
		Level                    string `yaml:"level"`
		HTTPAccessLogEnabled     *bool  `yaml:"http_access_log_enabled"`
		HTTPSlowRequestThreshold string `yaml:"http_slow_request_threshold"`
	} `yaml:"logging"`
	OpenAI struct {
		BaseURL string `yaml:"base_url"`
		APIKey  string `yaml:"api_key"`
		Model   string `yaml:"model"`
	} `yaml:"openai"`
	AI struct {
		PriceRule      string `yaml:"price_rule"`
		ConfidenceRule string `yaml:"confidence_rule"`
		ScoreRule      string `yaml:"score_rule"`
	} `yaml:"ai"`
	Market struct {
		CardmarketBaseURL         string         `yaml:"cardmarket_base_url"`
		CardmarketAppToken        string         `yaml:"cardmarket_app_token"`
		CardmarketAppSecret       string         `yaml:"cardmarket_app_secret"`
		CardmarketAccessToken     string         `yaml:"cardmarket_access_token"`
		CardmarketAccessSecret    string         `yaml:"cardmarket_access_token_secret"`
		CardmarketIDGame          int            `yaml:"cardmarket_id_game"`
		CardmarketHTTPTimeout     string         `yaml:"cardmarket_http_timeout"`
		CardmarketSinglesCacheTTL string         `yaml:"cardmarket_singles_cache_ttl"`
		TcgSetToExpansion         map[string]int `yaml:"tcg_set_to_expansion"`
	} `yaml:"market"`
	PokemonTCG struct {
		BaseURL                string `yaml:"base_url"`
		APIKey                 string `yaml:"api_key"`
		FallbackRequestsPerMin int    `yaml:"fallback_requests_per_min"`
	} `yaml:"pokemontcg"`
	MCP struct {
		Enable *bool `yaml:"enable"`
	} `yaml:"mcp"`
	Imageproc struct {
		CardNormalize       *bool   `yaml:"card_normalize"`
		StrictCardNormalize *bool   `yaml:"strict_card_normalize"`
		MaxWorkingLongEdge  int     `yaml:"max_working_long_edge"`
		WarpWidth           int     `yaml:"warp_width"`
		MinQuadAreaRatio    float64 `yaml:"min_quad_area_ratio"`
		MaxQuadAreaRatio    float64 `yaml:"max_quad_area_ratio"`
		DebugNormalize      struct {
			Enabled   *bool  `yaml:"enabled"`
			OutputDir string `yaml:"output_dir"`
		} `yaml:"debug_normalize"`
	} `yaml:"imageproc"`
}

func buildImageprocConfig(y yamlConfig) (imageproc.Config, error) {
	d := imageproc.DefaultConfig()
	if y.Imageproc.CardNormalize != nil {
		d.CardNormalize = *y.Imageproc.CardNormalize
	}
	if y.Imageproc.StrictCardNormalize != nil {
		d.StrictCardNormalize = *y.Imageproc.StrictCardNormalize
	}
	if y.Imageproc.MaxWorkingLongEdge > 0 {
		d.MaxWorkingLongEdge = y.Imageproc.MaxWorkingLongEdge
	}
	if y.Imageproc.WarpWidth > 0 {
		d.WarpWidth = y.Imageproc.WarpWidth
	}
	if y.Imageproc.MinQuadAreaRatio > 0 {
		d.MinQuadAreaRatio = y.Imageproc.MinQuadAreaRatio
	}
	if y.Imageproc.MaxQuadAreaRatio > 0 && y.Imageproc.MaxQuadAreaRatio <= 1 {
		d.MaxQuadAreaRatio = y.Imageproc.MaxQuadAreaRatio
	}
	if y.Imageproc.DebugNormalize.Enabled != nil {
		d.DebugNormalize = *y.Imageproc.DebugNormalize.Enabled
	}
	d.DebugNormalizeOutDir = y.Imageproc.DebugNormalize.OutputDir
	applyImageprocEnvOverrides(&d)
	if err := imageproc.ValidateDebug(d); err != nil {
		return imageproc.Config{}, err
	}
	return d, nil
}

// applyImageprocEnvOverrides applies optional env overrides after YAML (highest precedence).
// STRICT_CARD_DETECTION is an alias for STRICT_CARD_NORMALIZE (same as strict_card_normalize).
func applyImageprocEnvOverrides(d *imageproc.Config) {
	if v := strings.TrimSpace(os.Getenv("IMAGEPROC_CARD_NORMALIZE")); v != "" {
		if b, ok := parseEnvBool(v); ok {
			d.CardNormalize = b
		}
	}
	if v := strings.TrimSpace(os.Getenv("STRICT_CARD_NORMALIZE")); v != "" {
		if b, ok := parseEnvBool(v); ok {
			d.StrictCardNormalize = b
		}
	}
	if v := strings.TrimSpace(os.Getenv("STRICT_CARD_DETECTION")); v != "" {
		if b, ok := parseEnvBool(v); ok {
			d.StrictCardNormalize = b
		}
	}
	if v := strings.TrimSpace(os.Getenv("CARD_WARP_WIDTH")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 64 {
			d.WarpWidth = n
		}
	}
	if v := strings.TrimSpace(os.Getenv("IMAGEPROC_MAX_WORKING_LONG_EDGE")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 256 {
			d.MaxWorkingLongEdge = n
		}
	}
	if v := strings.TrimSpace(os.Getenv("IMAGEPROC_MIN_QUAD_AREA_RATIO")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 && f < 1 {
			d.MinQuadAreaRatio = f
		}
	}
	if v := strings.TrimSpace(os.Getenv("IMAGEPROC_MAX_QUAD_AREA_RATIO")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 && f <= 1 {
			d.MaxQuadAreaRatio = f
		}
	}
}

func parseEnvBool(s string) (v bool, ok bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "t", "yes", "y", "on":
		return true, true
	case "0", "false", "f", "no", "n", "off":
		return false, true
	default:
		return false, false
	}
}
