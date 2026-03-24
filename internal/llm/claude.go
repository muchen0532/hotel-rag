package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"hotel-rag/internal/config"
)

type claudeClient struct {
	cfg *config.LLMConfig
}

type claudeRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system"`
	Messages  []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *claudeClient) Ask(ctx context.Context, userMessage string) (string, error) {
	body, _ := json.Marshal(claudeRequest{
		Model:     c.cfg.Model,
		MaxTokens: c.cfg.MaxTokens,
		System:    SystemPrompt,
		Messages:  []message{{Role: "user", Content: userMessage}},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	if c.cfg.APIKey != "" {
		req.Header.Set("x-api-key", c.cfg.APIKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求Claude失败: %w", err)
	}
	defer resp.Body.Close()

	var result claudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Error != nil {
		return "", fmt.Errorf("claude错误: %s", result.Error.Message)
	}
	if len(result.Content) == 0 {
		return "", fmt.Errorf("claude返回空内容")
	}
	return result.Content[0].Text, nil
}
