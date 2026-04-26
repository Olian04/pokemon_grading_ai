package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pokemon_ai/internal/domain/grading"
	"pokemon_ai/internal/domain/imageproc"
	"pokemon_ai/internal/integrations/market"
	"pokemon_ai/internal/integrations/openai"
	"pokemon_ai/internal/integrations/pokemontcg"
	wm "pokemon_ai/internal/messaging/watermill"
	"pokemon_ai/internal/observability/metrics"
	httptransport "pokemon_ai/internal/transport/http"
	"pokemon_ai/internal/transport/http/handlers"
	mcptransport "pokemon_ai/internal/transport/mcp"

	"gopkg.in/yaml.v3"
)

type Config struct {
	HTTPAddr                 string
	ReadTimeout              time.Duration
	WriteTimeout             time.Duration
	LogLevel                 slog.Level
	HTTPAccessLogEnabled     bool
	HTTPSlowRequestThreshold time.Duration
	OpenAIBaseURL            string
	OpenAIAPIKey             string
	OpenAIModel              string
	AIPriceRule              string
	AIConfidenceRule         string
	AIScoreRule              string
	MarketCardmarketBaseURL  string
	MarketCardmarketAPIKey   string
	PokemonTCGBaseURL        string
	PokemonTCGAPIKey         string
	FallbackRequestsPerMin   int
	EnableMCP                bool
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
	bus, busCleanup, err := wm.NewInProcessBus()
	if err != nil {
		return nil, nil, err
	}

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

	service := grading.NewService(grading.Dependencies{
		AI:        aiClient,
		TCG:       tcgClient,
		Events:    bus,
		Analyzer:  imageproc.NewAnalyzer(),
		Market:    market.NewService(market.Config{CardmarketBaseURL: cfg.MarketCardmarketBaseURL, CardmarketAPIKey: cfg.MarketCardmarketAPIKey}),
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
		busCleanup()
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
		HTTPAddr:                 firstNonEmpty(y.HTTP.Addr, ":8080"),
		ReadTimeout:              parseDurationOrDefault(y.HTTP.ReadTimeout, 15*time.Second),
		WriteTimeout:             parseDurationOrDefault(y.HTTP.WriteTimeout, 60*time.Second),
		LogLevel:                 parseLevel(y.Logging.Level, slog.LevelInfo),
		HTTPAccessLogEnabled:     defaultBool(y.Logging.HTTPAccessLogEnabled, true),
		HTTPSlowRequestThreshold: parseDurationOrDefault(y.Logging.HTTPSlowRequestThreshold, 500*time.Millisecond),
		OpenAIBaseURL:            firstNonEmpty(y.OpenAI.BaseURL, "http://localhost:11434/v1"),
		OpenAIAPIKey:             y.OpenAI.APIKey,
		OpenAIModel:              firstNonEmpty(y.OpenAI.Model, "qwen2.5:7b"),
		AIPriceRule:              firstNonEmpty(y.AI.PriceRule, ">= 20"),
		AIConfidenceRule:         firstNonEmpty(y.AI.ConfidenceRule, "< 0.75"),
		AIScoreRule:              firstNonEmpty(y.AI.ScoreRule, ">= 7.5"),
		MarketCardmarketBaseURL:  firstNonEmpty(y.Market.CardmarketBaseURL, "https://api.cardmarket.com/ws/v2.0"),
		MarketCardmarketAPIKey:   y.Market.CardmarketAPIKey,
		PokemonTCGBaseURL:        firstNonEmpty(y.PokemonTCG.BaseURL, "https://api.pokemontcg.io/v2"),
		PokemonTCGAPIKey:         y.PokemonTCG.APIKey,
		FallbackRequestsPerMin:   defaultInt(y.PokemonTCG.FallbackRequestsPerMin, 15),
		EnableMCP:                defaultBool(y.MCP.Enable, false),
	}
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
		CardmarketBaseURL string `yaml:"cardmarket_base_url"`
		CardmarketAPIKey  string `yaml:"cardmarket_api_key"`
	} `yaml:"market"`
	PokemonTCG struct {
		BaseURL                string `yaml:"base_url"`
		APIKey                 string `yaml:"api_key"`
		FallbackRequestsPerMin int    `yaml:"fallback_requests_per_min"`
	} `yaml:"pokemontcg"`
	MCP struct {
		Enable *bool `yaml:"enable"`
	} `yaml:"mcp"`
}
