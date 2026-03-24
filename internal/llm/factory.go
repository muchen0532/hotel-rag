package llm

import (
	"fmt"

	"hotel-rag/internal/config"
)

// NewClient 根据配置的provider返回对应的LLM实现
// 新增模型只需要：1.实现Client接口 2.在这里加一个case
func NewClient(cfg *config.LLMConfig) (Client, error) {
	switch cfg.Provider {
	case "claude":
		return &claudeClient{cfg: cfg}, nil
	case "ollama":
		return &ollamaClient{cfg: cfg}, nil
	case "deepseek":
		return &deepseekClient{cfg: cfg}, nil
	default:
		return nil, fmt.Errorf("未知LLM provider: %q", cfg.Provider)
	}
}
