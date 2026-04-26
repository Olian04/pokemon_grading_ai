package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"pokemon_ai/internal/domain/grading"
)

type Config struct {
	BaseURL string
	APIKey  string
	Model   string
	Timeout time.Duration
}

type Client struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewClient(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) AssessSurface(ctx context.Context, req grading.AIAssistRequest) (grading.AIAssistResponse, error) {
	if c.baseURL == "" {
		return grading.AIAssistResponse{}, fmt.Errorf("openai base url is empty")
	}
	body := map[string]any{
		"model": c.model,
		"messages": []map[string]string{
			{"role": "system", "content": surfacePromptSystem},
			{"role": "user", "content": buildSurfacePrompt(req)},
		},
		"temperature": 0.1,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return grading.AIAssistResponse{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return grading.AIAssistResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return grading.AIAssistResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return grading.AIAssistResponse{}, fmt.Errorf("openai request failed with status %d", resp.StatusCode)
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return grading.AIAssistResponse{}, err
	}
	if len(out.Choices) == 0 {
		return grading.AIAssistResponse{}, fmt.Errorf("openai response had no choices")
	}
	return parseAIAssistResponse(out.Choices[0].Message.Content), nil
}
