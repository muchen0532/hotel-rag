package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"hotel-rag/internal/config"
)

type ollamaClient struct {
	cfg *config.LLMConfig
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

func (o *ollamaClient) Ask(ctx context.Context, userMessage string) (string, error) {
	body, _ := json.Marshal(ollamaRequest{
		Model: o.cfg.Model,
		Messages: []ollamaMessage{
			{Role: "system", Content: SystemPrompt},
			{Role: "user", Content: userMessage},
		},
		Stream: false,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.cfg.BaseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求Ollama失败，是否已运行 ollama serve: %w", err)
	}
	defer resp.Body.Close()

	var result ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Message.Content == "" {
		return "", fmt.Errorf("ollama返回空内容")
	}
	return result.Message.Content, nil
}
