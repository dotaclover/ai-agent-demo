package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"agent-demo/agent"
)

// RegisterTools 注册 AI 工具到 registry（闭包捕获 apiKey 等参数）
func RegisterTools(registry *agent.ToolRegistry, apiKey, baseURL, chatModel, imageModel, videoModel string) {
	// generate_image
	registry.Register(agent.NewTool(
		"generate_image",
		`AI 文生图，根据文字描述生成图片，支持自定义尺寸。注意：宽x高的像素总数不能低于 3686400（即至少 1920x1920）。
常见尺寸建议：16:9→2560x1440, 1:1→1920x1920, 9:16→1440x2560, 4:3→2560x1920, 3:4→1920x2560`,
		`{
			"type": "object",
			"properties": {
				"prompt": {"type": "string", "description": "图片描述（必填）"},
				"width": {"type": "integer", "description": "宽度像素，默认 2560"},
				"height": {"type": "integer", "description": "高度像素，默认 1440"},
				"guidance_scale": {"type": "number", "description": "引导系数 1-10，默认 3.0"}
			},
			"required": ["prompt"]
		}`,
		func(ctx context.Context, arguments string) (string, error) {
			var args struct {
				Prompt        string  `json:"prompt"`
				Width         int     `json:"width"`
				Height        int     `json:"height"`
				GuidanceScale float64 `json:"guidance_scale"`
			}
			if err := json.Unmarshal([]byte(arguments), &args); err != nil {
				return "", fmt.Errorf("invalid arguments: %w", err)
			}

			width := args.Width
			if width <= 0 {
				width = 2560
			}
			height := args.Height
			if height <= 0 {
				height = 1440
			}
			size := fmt.Sprintf("%dx%d", width, height)

			resp, err := CallImageGeneration(apiKey, baseURL, imageModel, args.Prompt, size, args.GuidanceScale)
			if err != nil {
				return "", err
			}

			result, _ := json.Marshal(map[string]interface{}{
				"image_url": resp.URL,
				"prompt":    args.Prompt,
				"size":      size,
				"message":   "图片生成成功",
			})
			return string(result), nil
		},
		false,
	))

	// generate_video
	registry.Register(agent.NewTool(
		"generate_video",
		"AI 文生视频（异步），提交后返回任务ID，需用 query_video_task 查询进度。可选提供首帧图片 URL 实现图生视频（使用 generate_image 返回的 image_url）。",
		`{
			"type": "object",
			"properties": {
				"prompt": {"type": "string", "description": "视频描述（必填）"},
				"image_url": {"type": "string", "description": "首帧图片URL（可选，图生视频）"},
				"duration": {"type": "integer", "description": "时长秒数，默认 5"}
			},
			"required": ["prompt"]
		}`,
		func(ctx context.Context, arguments string) (string, error) {
			var args struct {
				Prompt   string `json:"prompt"`
				ImageURL string `json:"image_url"`
				Duration int    `json:"duration"`
			}
			if err := json.Unmarshal([]byte(arguments), &args); err != nil {
				return "", fmt.Errorf("invalid arguments: %w", err)
			}

			content := []interface{}{
				map[string]interface{}{"type": "text", "text": args.Prompt},
			}
			if args.ImageURL != "" {
				content = append(content, map[string]interface{}{
					"type":      "image_url",
					"image_url": map[string]string{"url": args.ImageURL},
				})
			}

			resp, err := CallVideoGeneration(apiKey, baseURL, videoModel, content, args.Duration)
			if err != nil {
				return "", err
			}

			result, _ := json.Marshal(map[string]interface{}{
				"task_id": resp.TaskID,
				"status":  resp.Status,
				"prompt":   args.Prompt,
				"message": "视频生成任务已提交，请使用 query_video_task 查询进度",
			})
			return string(result), nil
		},
		false,
	))

	// write_article
	registry.Register(agent.NewTool(
		"write_article",
		"自媒体文章写手，根据主题撰写 1000 字以内的自媒体风格文章。返回纯文本文章内容。",
		`{
			"type": "object",
			"properties": {
				"topic": {"type": "string", "description": "文章主题或标题（必填）"},
				"style": {"type": "string", "description": "写作风格，如：轻松幽默、专业严谨、情感共鸣、干货分享，默认轻松幽默"},
				"length": {"type": "string", "description": "字数要求，如 '200字', '200-500字'，默认 '300字左右'"},
				"keywords": {"type": "string", "description": "需要包含的关键词（可选，逗号分隔）"}
			},
			"required": ["topic"]
		}`,
		func(ctx context.Context, arguments string) (string, error) {
			var args struct {
				Topic    string `json:"topic"`
				Style    string `json:"style"`
				Length   string `json:"length"`
				Keywords string `json:"keywords"`
			}
			if err := json.Unmarshal([]byte(arguments), &args); err != nil {
				return "", fmt.Errorf("invalid arguments: %w", err)
			}

			style := args.Style
			if style == "" {
				style = "轻松幽默"
			}

			length := args.Length
			if length == "" {
				length = "300字左右"
			}

			prompt := fmt.Sprintf(`请围绕主题「%s」撰写一篇自媒体文章。
要求：
- 风格：%s
- 字数：%s
- 包含吸引人的标题
- 结构清晰，分段合理
- 适合社交媒体发布`, args.Topic, style)
			if args.Keywords != "" {
				prompt += fmt.Sprintf("\n- 自然融入关键词：%s", args.Keywords)
			}

			// 用同一个 LLM 生成文章
			provider := agent.NewOpenAIProvider(agent.OpenAIProviderConfig{
				Name:    "doubao-writer",
				APIURL:  baseURL + "/chat/completions",
				APIKey:  apiKey,
				Model:   chatModel,
				Timeout: 60 * time.Second,
			})

			msgs := []agent.Message{
				{Role: agent.RoleSystem, Content: "你是一位资深自媒体写手，擅长撰写吸引读者的文章。只输出文章内容，不要输出其他内容。"},
				{Role: agent.RoleUser, Content: prompt},
			}
			resp, err := provider.Chat(ctx, msgs, nil, &agent.LLMConfig{Temperature: 0.8, MaxTokens: 2048})
			if err != nil {
				return "", fmt.Errorf("生成文章失败: %w", err)
			}

			result, _ := json.Marshal(map[string]interface{}{
				"article": resp.Content,
				"prompt":  prompt,
				"message": "文章生成成功",
			})
			return string(result), nil
		},
		false,
	))

	// query_video_task
	registry.Register(agent.NewTool(
		"query_video_task",
		"查询视频生成任务的进度。工具会自动每10秒轮询一次，最多等待3分钟，直到任务完成或失败才返回结果。",
		`{
			"type": "object",
			"properties": {
				"task_id": {"type": "string", "description": "视频任务ID（必填）"}
			},
			"required": ["task_id"]
		}`,
		func(ctx context.Context, arguments string) (string, error) {
			var args struct {
				TaskID string `json:"task_id"`
			}
			if err := json.Unmarshal([]byte(arguments), &args); err != nil {
				return "", fmt.Errorf("invalid arguments: %w", err)
			}

			maxWait := 3 * time.Minute
			interval := 10 * time.Second
			start := time.Now()

			for {
				resp, err := CallVideoQuery(apiKey, baseURL, args.TaskID)
				if err != nil {
					return "", err
				}

				if resp.Status == "succeeded" || resp.Status == "failed" {
					resultMap := map[string]interface{}{
						"task_id": resp.TaskID,
						"status":  resp.Status,
					}
					if resp.URL != "" {
						resultMap["video_url"] = resp.URL
					}
					if resp.Error != "" {
						resultMap["error"] = resp.Error
					}
					result, _ := json.Marshal(resultMap)
					return string(result), nil
				}

				if time.Since(start) >= maxWait {
					resultMap := map[string]interface{}{
						"task_id": resp.TaskID,
						"status":  resp.Status,
						"message": "已等待3分钟仍未完成，请稍后再查询",
					}
					result, _ := json.Marshal(resultMap)
					return string(result), nil
				}

				log.Printf("[Agent] 视频任务 %s 状态: %s，%ds 后重试", args.TaskID, resp.Status, int(interval.Seconds()))
				select {
				case <-ctx.Done():
					return "", fmt.Errorf("查询被取消")
				case <-time.After(interval):
				}
			}
		},
		false,
	))
}

// RegisterSearchTool 注册搜索工具（仅当 bochaKey 非空时调用）
func RegisterSearchTool(registry *agent.ToolRegistry, bochaKey string) {
	if bochaKey == "" {
		return
	}

	registry.Register(agent.NewTool(
		"web_search",
		"联网搜索工具，可以搜索互联网获取实时信息、新闻、数据等。当用户询问需要联网才能回答的问题时使用此工具。",
		`{
			"type": "object",
			"properties": {
				"query": {"type": "string", "description": "搜索关键词（必填）"},
				"count": {"type": "integer", "description": "返回结果数量，默认 8，最大 10"},
				"freshness": {"type": "string", "description": "时效性过滤：noLimit(默认)、day、week、month"}
			},
			"required": ["query"]
		}`,
		func(ctx context.Context, arguments string) (string, error) {
			var args struct {
				Query     string `json:"query"`
				Count     int    `json:"count"`
				Freshness string `json:"freshness"`
			}
			if err := json.Unmarshal([]byte(arguments), &args); err != nil {
				return "", fmt.Errorf("invalid arguments: %w", err)
			}

			count := args.Count
			if count <= 0 || count > 10 {
				count = 8
			}
			freshness := args.Freshness
			if freshness == "" {
				freshness = "noLimit"
			}

			reqBody, _ := json.Marshal(map[string]interface{}{
				"query":     args.Query,
				"summary":   true,
				"freshness": freshness,
				"count":     count,
			})

			req, _ := http.NewRequest("POST", "https://api.bocha.cn/v1/web-search", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+bochaKey)

			log.Printf("[Bocha] 搜索: %s", args.Query)

			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				return "", fmt.Errorf("搜索请求失败: %w", err)
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != http.StatusOK {
				log.Printf("[Bocha] 搜索失败 status=%d body=%s", resp.StatusCode, string(body))
				return "", fmt.Errorf("搜索 API 错误 (status %d): %s", resp.StatusCode, string(body))
			}

			// 解析博查响应
			var bochaResp struct {
				Code int `json:"code"`
				Data struct {
					WebPages struct {
						Value []struct {
							Name    string `json:"name"`
							URL     string `json:"url"`
							Snippet string `json:"snippet"`
						} `json:"value"`
					} `json:"webPages"`
				} `json:"data"`
			}
			if err := json.Unmarshal(body, &bochaResp); err != nil {
				return "", fmt.Errorf("解析搜索结果失败: %w", err)
			}

			// 格式化结果给 LLM
			type searchItem struct {
				Title   string `json:"title"`
				URL     string `json:"url"`
				Snippet string `json:"snippet"`
			}
			items := make([]searchItem, 0, len(bochaResp.Data.WebPages.Value))
			for _, v := range bochaResp.Data.WebPages.Value {
				if v.Name == "" && v.Snippet == "" {
					continue
				}
				items = append(items, searchItem{
					Title:   v.Name,
					URL:     v.URL,
					Snippet: v.Snippet,
				})
			}

			result, _ := json.Marshal(map[string]interface{}{
				"query":   args.Query,
				"results": items,
				"count":   len(items),
				"message": fmt.Sprintf("搜索到 %d 条结果", len(items)),
			})
			return string(result), nil
		},
		false,
	))
}
