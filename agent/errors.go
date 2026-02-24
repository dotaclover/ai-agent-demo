package agent

import "fmt"

// ErrMaxIterations 达到最大迭代次数
type ErrMaxIterations struct {
	Iterations int
}

func (e *ErrMaxIterations) Error() string {
	return fmt.Sprintf("agent reached max iterations (%d)", e.Iterations)
}

// ErrToolNotFound 工具不存在
type ErrToolNotFound struct {
	Name string
}

func (e *ErrToolNotFound) Error() string {
	return fmt.Sprintf("tool not found: %s", e.Name)
}

// ErrTooManyToolCalls 单轮工具调用过多
type ErrTooManyToolCalls struct {
	Count int
	Max   int
}

func (e *ErrTooManyToolCalls) Error() string {
	return fmt.Sprintf("too many tool calls in one turn: %d (max %d)", e.Count, e.Max)
}
