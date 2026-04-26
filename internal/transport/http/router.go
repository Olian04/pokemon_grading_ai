package http

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"pokemon_ai/internal/observability/metrics"
	"pokemon_ai/internal/transport/http/handlers"
	mcptransport "pokemon_ai/internal/transport/mcp"
)

type RouterConfig struct {
	Handlers             *handlers.Handler
	EnableMCP            bool
	MCPServer            *mcptransport.Server
	AccessLogEnabled     bool
	SlowRequestThreshold time.Duration
	Metrics              *metrics.Registry
}

func NewRouter(cfg RouterConfig) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", cfg.Handlers.Health)
	mux.HandleFunc("POST /v1/grade", cfg.Handlers.Grade)
	mux.HandleFunc("GET /v1/cards/search", cfg.Handlers.CardSearch)
	mux.HandleFunc("GET /v1/cards/pricing/{id}", cfg.Handlers.CardPricing)
	if cfg.Metrics != nil {
		mux.Handle("GET /metrics", cfg.Metrics.Handler())
	}
	if cfg.EnableMCP {
		mux.HandleFunc("POST /mcp", cfg.MCPServer.ServeHTTP)
	}
	return loggingAndMetricsMiddleware(mux, cfg.Metrics, cfg.AccessLogEnabled, cfg.SlowRequestThreshold)
}

func loggingAndMetricsMiddleware(next http.Handler, registry *metrics.Registry, accessLogEnabled bool, slowThreshold time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if registry != nil {
			registry.HTTP().IncInFlight()
			defer registry.HTTP().DecInFlight()
		}
		start := time.Now()
		rw := metrics.NewStatusWriter(w)

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		route := strings.TrimSpace(r.Pattern)
		if route == "" {
			route = "unmatched"
		}
		if registry != nil {
			registry.HTTP().ObserveRequest(route, rw.StatusCode(), duration)
		}

		if duration >= slowThreshold {
			slog.Warn("http_slow_request",
				"method", r.Method,
				"route", route,
				"status", rw.StatusCode(),
				"duration_ms", duration.Milliseconds(),
			)
			return
		}
		if accessLogEnabled {
			slog.Info("http_access",
				"method", r.Method,
				"route", route,
				"status", rw.StatusCode(),
				"duration_ms", duration.Milliseconds(),
			)
		}
	})
}
