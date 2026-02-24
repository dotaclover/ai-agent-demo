package agent

import "context"

// LLMProvider LLM 提供商接口
type LLMProvider interface {
	// Chat 发送消息获取回复
	Chat(ctx context.Context, messages []Message, tools []*Tool, cfg *LLMConfig) (*LLMResponse, error)
	// Name 提供商名称
	Name() string
	// SupportsTools 是否原生支持 function calling
	SupportsTools() bool
}

// LLMConfig LLM 调用配置
type LLMConfig struct {
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
}

// LLMResponse LLM 响应
type LLMResponse struct {
	Content          string     `json:"content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	PromptTokens     int        `json:"prompt_tokens"`
	CompletionTokens int        `json:"completion_tokens"`
}
