package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	defaultBaseURL    = "https://ark.cn-beijing.volces.com/api/v3"
	defaultImageModel = "doubao-seedream-4-5-251128"
	defaultVideoModel = "doubao-seedance-1-5-pro-251215"

	pathImageGen = "/images/generations"
	pathVideoGen = "/contents/generations/tasks"
)

var httpClient = &http.Client{Timeout: 120 * time.Second}

// ImageGenResult 图片生成结果
type ImageGenResult struct {
	URL string `json:"url"` // 公网 URL
}

// CallImageGeneration 调用豆包图片生成 API，返回公网 URL
func CallImageGeneration(apiKey, baseURL, model, prompt, size string, guidanceScale float64) (*ImageGenResult, error) {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if model == "" {
		model = defaultImageModel
	}

	body := map[string]interface{}{
		"model":           model,
		"prompt":          prompt,
		"size":            size,
		"response_format": "url",
	}
	if guidanceScale > 0 {
		body["guidance_scale"] = guidanceScale
	}

	jsonData, _ := json.Marshal(body)
	url := baseURL + pathImageGen
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	log.Printf("[Doubao] 图片生成: model=%s size=%s", model, size)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用 API 失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Printf("[Doubao] 图片生成失败 status=%d body=%s", resp.StatusCode, string(respBody))
		return nil, fmt.Errorf("API 错误 (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data []struct {
			URL string `json:"url"`
		} `json:"data"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	if result.Error != nil {
		log.Printf("[Doubao] 图片生成 API 错误: %s", result.Error.Message)
		return nil, fmt.Errorf("API 错误: %s", result.Error.Message)
	}
	if len(result.Data) == 0 || result.Data[0].URL == "" {
		return nil, fmt.Errorf("API 未返回图片")
	}

	return &ImageGenResult{URL: result.Data[0].URL}, nil
}

// VideoGenResult 视频生成结果
type VideoGenResult struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
}

// CallVideoGeneration 调用豆包视频生成 API
func CallVideoGeneration(apiKey, baseURL, model string, content []interface{}, duration int) (*VideoGenResult, error) {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if model == "" {
		model = defaultVideoModel
	}

	body := map[string]interface{}{
		"model":   model,
		"content": content,
	}
	if duration > 0 {
		body["duration"] = duration
	}

	jsonData, _ := json.Marshal(body)
	url := baseURL + pathVideoGen
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	log.Printf("[Doubao] 视频生成: model=%s", model)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用 API 失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Printf("[Doubao] 视频生成失败 status=%d body=%s", resp.StatusCode, string(respBody))
		return nil, fmt.Errorf("API 错误 (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("API 错误: %s", result.Error.Message)
	}

	return &VideoGenResult{
		TaskID: result.ID,
		Status: result.Status,
	}, nil
}

// VideoQueryResult 视频查询结果
type VideoQueryResult struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
	URL    string `json:"url,omitempty"`
	Error  string `json:"error,omitempty"`
}

// CallVideoQuery 查询视频任务状态
func CallVideoQuery(apiKey, baseURL, taskID string) (*VideoQueryResult, error) {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	url := fmt.Sprintf("%s%s/%s", baseURL, pathVideoGen, taskID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("查询任务失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		ID      string `json:"id"`
		Status  string `json:"status"`
		Content *struct {
			VideoURL string `json:"video_url"`
		} `json:"content,omitempty"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	response := &VideoQueryResult{
		TaskID: result.ID,
		Status: result.Status,
	}
	if result.Error != nil {
		response.Error = result.Error.Message
	}
	if result.Status == "succeeded" && result.Content != nil && result.Content.VideoURL != "" {
		response.URL = result.Content.VideoURL
	}

	return response, nil
}
