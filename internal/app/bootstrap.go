package app

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"pokemon_ai/internal/domain/grading"
	"pokemon_ai/internal/domain/imageproc"
	"pokemon_ai/internal/integrations/openai"
	"pokemon_ai/internal/integrations/pokemontcg"
	wm "pokemon_ai/internal/messaging/watermill"
	httptransport "pokemon_ai/internal/transport/http"
	"pokemon_ai/internal/transport/http/handlers"
	mcptransport "pokemon_ai/internal/transport/mcp"
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
	PokemonTCGBaseURL        string
	PokemonTCGAPIKey         string
	FallbackRequestsPerMin   int
	EnableMCP                bool
}

func Bootstrap(ctx context.Context) (*http.Server, func(), error) {
	cfg := loadConfig()

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
		AI:       aiClient,
		TCG:      tcgClient,
		Events:   bus,
		Analyzer: imageproc.NewAnalyzer(),
	})

	h := handlers.New(handlers.Dependencies{
		Grading: service,
		TCG:     tcgClient,
	})

	router := httptransport.NewRouter(httptransport.RouterConfig{
		Handlers:             h,
		EnableMCP:            cfg.EnableMCP,
		MCPServer:            mcptransport.NewServer(service),
		AccessLogEnabled:     cfg.HTTPAccessLogEnabled,
		SlowRequestThreshold: cfg.HTTPSlowRequestThreshold,
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

func loadConfig() Config {
	return Config{
		HTTPAddr:                 getenv("HTTP_ADDR", ":8080"),
		ReadTimeout:              durationEnv("HTTP_READ_TIMEOUT", 15*time.Second),
		WriteTimeout:             durationEnv("HTTP_WRITE_TIMEOUT", 60*time.Second),
		LogLevel:                 logLevelEnv("LOG_LEVEL", slog.LevelInfo),
		HTTPAccessLogEnabled:     boolEnv("HTTP_ACCESS_LOG_ENABLED", true),
		HTTPSlowRequestThreshold: durationEnv("HTTP_SLOW_REQUEST_THRESHOLD", 500*time.Millisecond),
		OpenAIBaseURL:            getenv("OPENAI_BASE_URL", "http://localhost:11434/v1"),
		OpenAIAPIKey:             os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:              getenv("OPENAI_MODEL", "qwen2.5:7b"),
		PokemonTCGBaseURL:        getenv("POKEMON_TCG_BASE_URL", "https://api.pokemontcg.io/v2"),
		PokemonTCGAPIKey:         os.Getenv("POKEMON_TCG_API_KEY"),
		FallbackRequestsPerMin:   intEnv("POKEMON_TCG_FALLBACK_RPM", 15),
		EnableMCP:                boolEnv("ENABLE_MCP", false),
	}
}

func logLevelEnv(key string, fallback slog.Level) slog.Level {
	raw := os.Getenv(key)
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

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func intEnv(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func boolEnv(key string, fallback bool) bool {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return v
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return v
}
