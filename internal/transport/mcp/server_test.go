package mcp

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"pokemon_ai/internal/domain/grading"
)

type fakeGrader struct{}

func (fakeGrader) GradeCard(_ context.Context, _ grading.GradeRequest) (grading.GradeResponse, error) {
	return grading.GradeResponse{
		OverallProxy1To10: 8.1,
		SellerCondition:   "NM",
	}, nil
}

func TestToolsList(t *testing.T) {
	s := NewServer(fakeGrader{})
	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"grade_card"`)) {
		t.Fatalf("expected grade_card in response: %s", w.Body.String())
	}
}
