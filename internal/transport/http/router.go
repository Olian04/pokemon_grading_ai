package http

import (
	"log/slog"
	"net/http"
	"time"

	"pokemon_ai/internal/transport/http/handlers"
	mcptransport "pokemon_ai/internal/transport/mcp"
)

type RouterConfig struct {
	Handlers             *handlers.Handler
	EnableMCP            bool
	MCPServer            *mcptransport.Server
	AccessLogEnabled     bool
	SlowRequestThreshold time.Duration
}

func NewRouter(cfg RouterConfig) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", cfg.Handlers.Health)
	mux.HandleFunc("/v1/grade", cfg.Handlers.Grade)
	mux.HandleFunc("/v1/cards/search", cfg.Handlers.CardSearch)
	mux.HandleFunc("/v1/cards/pricing/", cfg.Handlers.CardPricing)
	if cfg.EnableMCP {
		mux.HandleFunc("/mcp", cfg.MCPServer.ServeHTTP)
	}
	return loggingMiddleware(mux, cfg.AccessLogEnabled, cfg.SlowRequestThreshold)
}

func loggingMiddleware(next http.Handler, accessLogEnabled bool, slowThreshold time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		if duration >= slowThreshold {
			slog.Warn("http_slow_request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.statusCode,
				"duration_ms", duration.Milliseconds(),
			)
			return
		}
		if accessLogEnabled {
			slog.Info("http_access",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.statusCode,
				"duration_ms", duration.Milliseconds(),
			)
		}
	})
}

type statusWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
