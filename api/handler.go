package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"agent-demo/agent"
)

const maxMessages = 100

// Session 内存会话
type Session struct {
	ID       string          `json:"id"`
	Messages []agent.Message `json:"messages"`
	Created  time.Time       `json:"created_at"`
}

var sessions sync.Map

func newID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Handler HTTP 处理器
type Handler struct {
	chatModel string
	baseURL   string
}

// NewHandler 创建 Handler
func NewHandler() *Handler {
	chatModel := os.Getenv("ARK_CHAT_MODEL")
	if chatModel == "" {
		chatModel = "doubao-seed-2-0-pro-260215"
	}
	baseURL := os.Getenv("ARK_BASE_URL")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Handler{
		chatModel: chatModel,
		baseURL:   baseURL,
	}
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/chat", h.handleChat)
	mux.HandleFunc("POST /api/reset", h.handleReset)
}

// chatRequest 聊天请求
type chatRequest struct {
	Message   string `json:"message"`
	APIKey    string `json:"api_key"`
	BochaKey  string `json:"bocha_key"`
	SessionID string `json:"session_id"`
}

// chatResponse 聊天响应
type chatResponse struct {
	SessionID string          `json:"session_id"`
	Messages  []agent.Message `json:"messages"`
}

func (h *Handler) handleChat(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效的请求"})
		return
	}
	if req.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "消息不能为空"})
		return
	}
	if req.APIKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请提供 API Key"})
		return
	}

	// 获取或创建会话
	var session *Session
	if req.SessionID != "" {
		if v, ok := sessions.Load(req.SessionID); ok {
			session = v.(*Session)
		}
	}
	if session == nil {
		session = &Session{
			ID:      newID(),
			Created: time.Now(),
		}
	}

	// 创建 LLM Provider
	provider := agent.NewOpenAIProvider(agent.OpenAIProviderConfig{
		Name:    "doubao",
		APIURL:  h.baseURL + "/chat/completions",
		APIKey:  req.APIKey,
		Model:   h.chatModel,
		Timeout: 120 * time.Second,
	})

	// 创建工具注册表
	registry := agent.NewToolRegistry()
	imageModel := os.Getenv("ARK_IMAGE_MODEL")
	if imageModel == "" {
		imageModel = defaultImageModel
	}
	videoModel := os.Getenv("ARK_VIDEO_MODEL")
	if videoModel == "" {
		videoModel = defaultVideoModel
	}
	RegisterTools(registry, req.APIKey, h.baseURL, h.chatModel, imageModel, videoModel)
	RegisterSearchTool(registry, req.BochaKey)

	// 构建历史消息（排除 system）
	userMsg := agent.Message{
		ID:        newID(),
		Role:      agent.RoleUser,
		Content:   req.Message,
		CreatedAt: time.Now(),
	}
	historyMessages := make([]agent.Message, 0, len(session.Messages)+1)
	for _, m := range session.Messages {
		if m.Role != agent.RoleSystem {
			historyMessages = append(historyMessages, m)
		}
	}
	historyMessages = append(historyMessages, userMsg)

	// 设置 SSE 响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// 发送初始 SessionID
	sessionID := session.ID
	fmt.Fprintf(w, "event: session\ndata: %s\n\n", sessionID)
	flusher.Flush()

	// 运行 Agent
	ag := agent.New(provider, registry, nil)
	result, err := ag.Run(context.Background(), historyMessages, func(m agent.Message) {
		// 只发送新的非 system 消息
		if m.Role != agent.RoleSystem {
			data, _ := json.Marshal(m)
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", string(data))
			flusher.Flush()
		}
	})

	if err != nil {
		log.Printf("[Chat] Agent error: %v", err)
		errMsg := map[string]string{"error": fmt.Sprintf("Agent 错误: %v", err)}
		data, _ := json.Marshal(errMsg)
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", string(data))
		flusher.Flush()
	}

	// 提取并保存最终消息（跳过 system）
	var newMessages []agent.Message
	if result != nil {
		for _, m := range result.Messages {
			if m.Role != agent.RoleSystem {
				newMessages = append(newMessages, m)
			}
		}
	}

	// 保留最近 maxMessages 条
	if len(newMessages) > maxMessages {
		newMessages = newMessages[len(newMessages)-maxMessages:]
	}

	session.Messages = newMessages
	sessions.Store(session.ID, session)

	// 发送结束标识
	fmt.Fprintf(w, "event: done\ndata: {}\n\n")
	flusher.Flush()
}

func (h *Handler) handleReset(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID string `json:"session_id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.SessionID != "" {
		sessions.Delete(req.SessionID)
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "已重置"})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
