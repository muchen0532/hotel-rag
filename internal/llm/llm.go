package llm

import "context"

const SystemPrompt = `你是一个酒店入住率分析助手，数据来自2025年全年。
根据提供的统计摘要和检索数据直接回答用户问题，给出具体数字和结论。
不要要求用户补充信息，用现有数据尽力分析。回答简洁专业。`

// Client 是所有LLM实现必须满足的接口
// handler只依赖这个接口，不依赖任何具体实现
type Client interface {
	Ask(ctx context.Context, message string) (string, error)
}
