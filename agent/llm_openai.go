package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// OpenAIProvider OpenAI 兼容的 LLM Provider（豆包、DeepSeek 等）
type OpenAIProvider struct {
	name       string
	apiURL     string
	apiKey     string
	model      string
	httpClient *http.Client
}

// OpenAIProviderConfig OpenAI Provider 配置
type OpenAIProviderConfig struct {
	Name    string
	APIURL  string
	APIKey  string
	Model   string
	Timeout time.Duration
}

// NewOpenAIProvider 创建 OpenAI 兼容 Provider
func NewOpenAIProvider(cfg OpenAIProviderConfig) *OpenAIProvider {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}
	return &OpenAIProvider{
		name:   cfg.Name,
		apiURL: cfg.APIURL,
		apiKey: cfg.APIKey,
		model:  cfg.Model,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (p *OpenAIProvider) Name() string        { return p.name }
func (p *OpenAIProvider) SupportsTools() bool { return true }

// Chat 调用 OpenAI 兼容的 chat completions API
func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message, tools []*Tool, cfg *LLMConfig) (*LLMResponse, error) {
	reqMessages := make([]openAIMessage, 0, len(messages))
	for _, m := range messages {
		msg := openAIMessage{
			Role:    string(m.Role),
			Content: m.Content,
		}
		if m.ToolCallID != "" {
			msg.ToolCallID = m.ToolCallID
		}
		if m.Name != "" {
			msg.Name = m.Name
		}
		if len(m.ToolCalls) > 0 {
			msg.ToolCalls = make([]openAIToolCall, len(m.ToolCalls))
			for i, tc := range m.ToolCalls {
				msg.ToolCalls[i] = openAIToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: openAIFunction{
						Name:      tc.Name,
						Arguments: tc.Arguments,
					},
				}
			}
		}
		reqMessages = append(reqMessages, msg)
	}

	reqBody := openAIChatRequest{
		Model:       p.model,
		Messages:    reqMessages,
		Temperature: cfg.Temperature,
		MaxTokens:   cfg.MaxTokens,
	}

	if len(tools) > 0 {
		reqTools := make([]openAITool, len(tools))
		for i, t := range tools {
			var params ParameterSchema
			json.Unmarshal([]byte(t.Parameters), &params)

			reqTools[i] = openAITool{
				Type: "function",
				Function: openAIToolDef{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  params,
				},
			}
		}
		reqBody.Tools = reqTools
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	startTime := time.Now()
	resp, err := p.httpClient.Do(req)
	elapsed := time.Since(startTime)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	log.Printf("[LLM] %s model=%s status=%d elapsed=%dms", p.name, p.model, resp.StatusCode, elapsed.Milliseconds())

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, truncate(string(body), 500))
	}

	var result openAIChatResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no response choices from %s", p.name)
	}

	choice := result.Choices[0]
	llmResp := &LLMResponse{
		Content:          choice.Message.Content,
		PromptTokens:     result.Usage.PromptTokens,
		CompletionTokens: result.Usage.CompletionTokens,
	}

	for _, tc := range choice.Message.ToolCalls {
		llmResp.ToolCalls = append(llmResp.ToolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	return llmResp, nil
}

// OpenAI API 请求/响应结构

type openAIChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Tools       []openAITool    `json:"tools,omitempty"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	Name       string           `json:"name,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAITool struct {
	Type     string        `json:"type"`
	Function openAIToolDef `json:"function"`
}

type openAIToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  ParameterSchema `json:"parameters"`
}

type openAIToolCall struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAIChatResponse struct {
	Choices []openAIChoice `json:"choices"`
	Usage   openAIUsage    `json:"usage"`
}

type openAIChoice struct {
	Message openAIMessage `json:"message"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
