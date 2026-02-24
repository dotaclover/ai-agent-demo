package agent

import "context"

// ToolFunc 工具执行函数
type ToolFunc func(ctx context.Context, arguments string) (string, error)

// Tool 工具定义
type Tool struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Parameters  string   `json:"parameters"` // JSON Schema 字符串
	Execute     ToolFunc `json:"-"`
	Destructive bool     `json:"-"` // 需要用户确认
}

// NewTool 创建工具
func NewTool(name, description, parameters string, execute ToolFunc, destructive bool) *Tool {
	return &Tool{
		Name:        name,
		Description: description,
		Parameters:  parameters,
		Execute:     execute,
		Destructive: destructive,
	}
}

// ParameterSchema JSON Schema 描述工具参数
type ParameterSchema struct {
	Type       string                    `json:"type"` // "object"
	Properties map[string]PropertySchema `json:"properties"`
	Required   []string                  `json:"required,omitempty"`
}

// PropertySchema 单个参数的 JSON Schema
type PropertySchema struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

// ToolRegistry 工具注册表
type ToolRegistry struct {
	tools map[string]*Tool
	order []string // 保持注册顺序
}

// NewToolRegistry 创建工具注册表
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]*Tool),
	}
}

// Register 注册工具
func (r *ToolRegistry) Register(tool *Tool) {
	r.tools[tool.Name] = tool
	r.order = append(r.order, tool.Name)
}

// Get 获取工具
func (r *ToolRegistry) Get(name string) (*Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// List 列出所有工具（按注册顺序）
func (r *ToolRegistry) List() []*Tool {
	result := make([]*Tool, 0, len(r.order))
	for _, name := range r.order {
		if t, ok := r.tools[name]; ok {
			result = append(result, t)
		}
	}
	return result
}
