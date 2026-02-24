package agent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// newID 生成随机 ID（替代 uuid）
func newID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Agent AI 智能体
type Agent struct {
	provider LLMProvider
	registry *ToolRegistry
	config   *AgentConfig
}

// New 创建 Agent
func New(provider LLMProvider, registry *ToolRegistry, config *AgentConfig) *Agent {
	if config == nil {
		config = DefaultConfig()
	}
	return &Agent{
		provider: provider,
		registry: registry,
		config:   config,
	}
}

// RunResult 运行结果
type RunResult struct {
	Messages         []Message `json:"messages"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	Iterations       int       `json:"iterations"`
}

// Run 执行 Agent 编排循环，支持通过 onMessage 回调实时返回中间消息
func (a *Agent) Run(ctx context.Context, messages []Message, onMessage func(Message)) (*RunResult, error) {
	ctx, cancel := context.WithTimeout(ctx, a.config.Timeout)
	defer cancel()

	// 构建完整消息列表（prepend system prompt）
	allMessages := make([]Message, 0, len(messages)+1)
	allMessages = append(allMessages, Message{
		ID:        newID(),
		Role:      RoleSystem,
		Content:   a.config.SystemPrompt,
		CreatedAt: time.Now(),
	})
	allMessages = append(allMessages, messages...)

	tools := a.registry.List()
	llmCfg := &LLMConfig{
		Temperature: a.config.Temperature,
		MaxTokens:   a.config.MaxTokens,
	}

	result := &RunResult{}

	for i := 0; i < a.config.MaxIterations; i++ {
		result.Iterations = i + 1

		resp, err := a.provider.Chat(ctx, allMessages, tools, llmCfg)
		if err != nil {
			return result, fmt.Errorf("LLM call failed at iteration %d: %w", i+1, err)
		}

		result.PromptTokens += resp.PromptTokens
		result.CompletionTokens += resp.CompletionTokens

		// 没有 tool_calls → 最终回复
		if len(resp.ToolCalls) == 0 {
			assistantMsg := Message{
				ID:        newID(),
				Role:      RoleAssistant,
				Content:   resp.Content,
				CreatedAt: time.Now(),
			}
			allMessages = append(allMessages, assistantMsg)
			if onMessage != nil {
				onMessage(assistantMsg)
			}
			result.Messages = allMessages
			return result, nil
		}

		// 检查单轮 tool_calls 数量
		if len(resp.ToolCalls) > a.config.MaxToolCallsPerTurn {
			return result, &ErrTooManyToolCalls{
				Count: len(resp.ToolCalls),
				Max:   a.config.MaxToolCallsPerTurn,
			}
		}

		// 添加 assistant 消息（包含 tool_calls）
		assistantMsg := Message{
			ID:        newID(),
			Role:      RoleAssistant,
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
			CreatedAt: time.Now(),
		}
		allMessages = append(allMessages, assistantMsg)
		if onMessage != nil {
			onMessage(assistantMsg)
		}

		// 执行每个 tool_call
		for _, tc := range resp.ToolCalls {
			tool, ok := a.registry.Get(tc.Name)
			if !ok {
				errMsg := Message{
					ID:         newID(),
					Role:       RoleTool,
					Content:    fmt.Sprintf("Error: tool '%s' not found", tc.Name),
					ToolCallID: tc.ID,
					Name:       tc.Name,
					CreatedAt:  time.Now(),
				}
				allMessages = append(allMessages, errMsg)
				if onMessage != nil {
					onMessage(errMsg)
				}
				continue
			}

			toolResult, err := tool.Execute(ctx, tc.Arguments)
			if err != nil {
				toolResult = fmt.Sprintf("Error executing %s: %s", tc.Name, err.Error())
			}

			toolMsg := Message{
				ID:         newID(),
				Role:       RoleTool,
				Content:    toolResult,
				ToolCallID: tc.ID,
				Name:       tc.Name,
				CreatedAt:  time.Now(),
			}
			allMessages = append(allMessages, toolMsg)
			if onMessage != nil {
				onMessage(toolMsg)
			}
		}
	}

	result.Messages = allMessages
	return result, &ErrMaxIterations{Iterations: a.config.MaxIterations}
}
