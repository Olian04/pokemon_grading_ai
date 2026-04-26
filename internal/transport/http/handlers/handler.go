package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"pokemon_ai/internal/domain/grading"
	"pokemon_ai/internal/integrations/pokemontcg"
)

type GradingService interface {
	GradeCard(ctx context.Context, req grading.GradeRequest) (grading.GradeResponse, error)
}

type PokemonTCGService interface {
	SearchCards(ctx context.Context, query string) ([]pokemontcg.Card, error)
	GetCardPricing(ctx context.Context, id string) (pokemontcg.PriceSummary, error)
}

type Dependencies struct {
	Grading GradingService
	TCG     PokemonTCGService
}

type Handler struct {
	grading GradingService
	tcg     PokemonTCGService
}

func New(deps Dependencies) *Handler {
	return &Handler{
		grading: deps.Grading,
		tcg:     deps.TCG,
	}
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

type gradeInput struct {
	FrontImagePath string `json:"front_image_path"`
	BackImagePath  string `json:"back_image_path"`
	CardNameHint   string `json:"card_name_hint"`
	SetCodeHint    string `json:"set_code_hint"`
	CardNumberHint string `json:"card_number_hint"`
}

func (h *Handler) Grade(w http.ResponseWriter, r *http.Request) {
	var in gradeInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	resp, err := h.grading.GradeCard(r.Context(), grading.GradeRequest{
		FrontImagePath: in.FrontImagePath,
		BackImagePath:  in.BackImagePath,
		CardNameHint:   in.CardNameHint,
		SetCodeHint:    in.SetCodeHint,
		CardNumberHint: in.CardNumberHint,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) CardSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		http.Error(w, "missing q query parameter", http.StatusBadRequest)
		return
	}
	cards, err := h.tcg.SearchCards(r.Context(), q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cards": cards})
}

func (h *Handler) CardPricing(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.Error(w, "missing card id", http.StatusBadRequest)
		return
	}
	price, err := h.tcg.GetCardPricing(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, price)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
