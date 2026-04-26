package mcp

import (
	"context"
	"encoding/json"
	"net/http"

	"pokemon_ai/internal/domain/grading"
)

type gradingService interface {
	GradeCard(ctx context.Context, req grading.GradeRequest) (grading.GradeResponse, error)
}

type Server struct {
	grading gradingService
}

func NewServer(gradingSvc gradingService) *Server {
	return &Server{grading: gradingSvc}
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json-rpc request", http.StatusBadRequest)
		return
	}
	switch req.Method {
	case "tools/list":
		writeResult(w, req.ID, map[string]any{
			"tools": []map[string]any{
				{
					"name":        "grade_card",
					"description": "Grade a pokemon card image",
				},
			},
		})
	case "tools/call":
		s.handleToolCall(w, req.ID, req.Params)
	default:
		writeError(w, req.ID, -32601, "method not found")
	}
}

func writeResult(w http.ResponseWriter, id any, result any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	})
}

func writeError(w http.ResponseWriter, id any, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
