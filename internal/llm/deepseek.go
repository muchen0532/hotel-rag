package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"hotel-rag/internal/config"
)

type deepseekClient struct {
	cfg *config.LLMConfig
}

type deepseekRequest struct {
	Model    string            `json:"model"`
	Stream   bool              `json:"stream"`
	Messages []deepseekMessage `json:"messages"`
}

type deepseekMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type deepseekResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (o *deepseekClient) Ask(ctx context.Context, userMessage string) (string, error) {
	body, _ := json.Marshal(deepseekRequest{
		Model:  o.cfg.Model,
		Stream: false,
		Messages: []deepseekMessage{
			{Role: "system", Content: SystemPrompt},
			{Role: "user", Content: userMessage},
		},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.cfg.BaseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if o.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+o.cfg.APIKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求deepseek失败: %w", err)
	}
	defer resp.Body.Close()

	var result deepseekResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	fmt.Println(result)
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("deepseek返回空内容")
	}
	return result.Choices[0].Message.Content, nil
}
