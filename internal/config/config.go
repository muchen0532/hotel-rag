package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
	LLM    LLMConfig    `yaml:"llm"`
	Data   DataConfig   `yaml:"data"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type LLMConfig struct {
	Provider  string `yaml:"provider"` // claude | ollama
	BaseURL   string `yaml:"base_url"`
	APIKey    string `yaml:"api_key"`
	Model     string `yaml:"model"`
	MaxTokens int    `yaml:"max_tokens"`
}

type DataConfig struct {
	CSVPath     string `yaml:"csv_path"`
	SummaryPath string `yaml:"summary_path"`
	TopK        int    `yaml:"top_k"`
}

func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("打开配置文件失败: %w", err)
	}
	defer f.Close()

	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// 环境变量可以覆盖配置文件（生产环境用）
	if key := os.Getenv("LLM_API_KEY"); key != "" {
		cfg.LLM.APIKey = key
	}

	return &cfg, nil
}
