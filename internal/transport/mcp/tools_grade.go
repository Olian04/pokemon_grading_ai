package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"pokemon_ai/internal/domain/grading"
)

func (s *Server) handleToolCall(w http.ResponseWriter, id any, params json.RawMessage) {
	var req struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		writeError(w, id, -32602, "invalid params")
		return
	}
	if req.Name != "grade_card" {
		writeError(w, id, -32602, "unsupported tool")
		return
	}
	var args grading.GradeRequest
	if err := json.Unmarshal(req.Arguments, &args); err != nil {
		writeError(w, id, -32602, "invalid tool arguments")
		return
	}
	resp, err := s.grading.GradeCard(context.Background(), args)
	if err != nil {
		writeError(w, id, -32000, fmt.Sprintf("grade failed: %v", err))
		return
	}
	writeResult(w, id, map[string]any{
		"content": []map[string]any{
			{
				"type": "json",
				"json": resp,
			},
		},
	})
}

var _ http.Handler = (*Server)(nil)
