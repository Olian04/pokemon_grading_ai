package http

import (
	"net/http"

	"pokemon_ai/internal/transport/http/handlers"
	mcptransport "pokemon_ai/internal/transport/mcp"
)

func NewRouter(h *handlers.Handler, enableMCP bool, mcpServer *mcptransport.Server) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", h.Health)
	mux.HandleFunc("/v1/grade", h.Grade)
	mux.HandleFunc("/v1/cards/search", h.CardSearch)
	mux.HandleFunc("/v1/cards/pricing/", h.CardPricing)
	if enableMCP {
		mux.HandleFunc("/mcp", mcpServer.ServeHTTP)
	}
	return mux
}
