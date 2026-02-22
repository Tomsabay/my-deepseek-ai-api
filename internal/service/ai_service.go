// Package service 包含核心业务逻辑
// 这里实现了与 AI 大模型的通信逻辑
package service

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"ai-api/internal/model"
)

// ============================================
// AIService AI 服务接口（扩展版）
// ============================================
// 支持单轮对话、多轮对话、流式响应和多模态（图片）
type AIService interface {
	// Chat 单轮对话（简单模式）
	Chat(message string) (string, error)

	// ChatWithHistory 多轮对话（传入完整对话历史）
	ChatWithHistory(messages []model.OpenAIMessage) (string, error)

	// ChatStream 流式对话（通过 channel 返回事件，区分思考过程与正式回答）
	ChatStream(messages []model.OpenAIMessage) (<-chan model.StreamEvent, <-chan error)

	// ChatWithMultimodal 多模态对话（支持图片，非流式）
	ChatWithMultimodal(messages []model.OpenAIMultimodalMessage) (string, error)

	// ChatStreamMultimodal 多模态流式对话（支持图片）
	ChatStreamMultimodal(messages []model.OpenAIMultimodalMessage) (<-chan model.StreamEvent, <-chan error)

	// GetSystemPrompt 获取系统提示词
	GetSystemPrompt() string
}

// ============================================
// OpenAIService OpenAI 兼容的 AI 服务实现
// ============================================
type OpenAIService struct {
	baseURL      string
	apiKey       string
	model        string
	systemPrompt string
	httpClient   *http.Client
}

// ============================================
// NewOpenAIService 创建一个新的 OpenAI 服务实例
// ============================================
func NewOpenAIService(baseURL, apiKey, modelName, systemPrompt string) *OpenAIService {
	return &OpenAIService{
		baseURL:      baseURL,
		apiKey:       apiKey,
		model:        modelName,
		systemPrompt: systemPrompt,
		httpClient: &http.Client{
			// AI 模型的长推理（尤其是 R1）可能长达十几分钟，设置 15 分钟防断流
			Timeout: 900 * time.Second,
		},
	}
}

// GetSystemPrompt 返回系统提示词
func (s *OpenAIService) GetSystemPrompt() string {
	return s.systemPrompt
}

// WithModel 返回一个使用指定模型名的克隆实例
// 这样 handler 可以在每次请求时动态切换模型，而不影响原始服务实例
// 例如：svc.WithModel("deepseek-reasoner").ChatStream(messages)
func (s *OpenAIService) WithModel(modelName string) *OpenAIService {
	if modelName == "" {
		return s // 未指定模型时，直接返回原实例
	}
	// 浅拷贝结构体，只修改 model 字段
	clone := *s
	clone.model = modelName
	return &clone
}

// ============================================
// Chat 单轮对话（保持向后兼容）
// ============================================
func (s *OpenAIService) Chat(message string) (string, error) {
	messages := []model.OpenAIMessage{
		{Role: "system", Content: s.systemPrompt},
		{Role: "user", Content: message},
	}
	return s.ChatWithHistory(messages)
}

// ============================================
// ChatWithHistory 多轮对话
// ============================================
// 参数：
//   - messages: 完整的对话历史（包含 system、user、assistant 消息）
//
// 返回：
//   - string: AI 的回复
//   - error: 错误信息
//
// 技术讲解：
// 这个方法接收完整的对话历史，让 AI 能够理解上下文
// 调用方负责维护对话历史，每次请求时传入完整历史
func (s *OpenAIService) ChatWithHistory(messages []model.OpenAIMessage) (string, error) {
	// 构建请求
	reqBody := model.OpenAIChatRequest{
		Model:       s.model,
		Messages:    messages,
		Temperature: 0.7,
		Stream:      false, // 非流式
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	// 发送请求
	url := s.baseURL + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API 返回错误状态码 %d: %s", resp.StatusCode, string(body))
	}

	var chatResp model.OpenAIChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("AI API 错误: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("AI 没有返回任何回复")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// ============================================
// ChatStream 流式对话
// ============================================
// 参数：
//   - messages: 完整的对话历史
//
// 返回：
//   - <-chan string: 内容数据块的 channel（每个 chunk 是一小段文本）
//   - <-chan error: 错误 channel
//
// 技术讲解：
// 使用 Go 的 channel 来实现异步流式传输
// - contentChan: 发送每个文本片段
// - errChan: 发送错误（如果有）
//
// 使用方式：
//
//	contentChan, errChan := service.ChatStream(messages)
//	for {
//	    select {
//	    case content, ok := <-contentChan:
//	        if !ok { return } // channel 关闭，结束
//	        fmt.Print(content) // 输出文本片段
//	    case err := <-errChan:
//	        if err != nil { handleError(err) }
//	    }
//	}
func (s *OpenAIService) ChatStream(messages []model.OpenAIMessage) (<-chan model.StreamEvent, <-chan error) {
	contentChan := make(chan model.StreamEvent, 100)
	errChan := make(chan error, 1)

	go func() {
		defer close(contentChan)
		defer close(errChan)

		// 构建请求（设置 stream: true）
		reqBody := model.OpenAIChatRequest{
			Model:       s.model,
			Messages:    messages,
			Temperature: 0.7,
			Stream:      true, // 启用流式响应
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			errChan <- fmt.Errorf("序列化请求失败: %w", err)
			return
		}

		// 发送请求
		start := time.Now()
		url := s.baseURL + "/chat/completions"
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			errChan <- fmt.Errorf("创建请求失败: %w", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
		req.Header.Set("Accept", "text/event-stream") // SSE 专用 header

		resp, err := s.httpClient.Do(req)
		if err != nil {
			errChan <- fmt.Errorf("发送请求失败: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errChan <- fmt.Errorf("API 返回错误状态码 %d: %s", resp.StatusCode, string(body))
			return
		}

		log.Printf("chat stream: upstream ok, status=%d, elapsed=%s", resp.StatusCode, time.Since(start))

		// 逐行读取 SSE 流
		// SSE 格式：每行以 "data: " 开头，以 "\n\n" 结束
		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break // 流结束
				}
				errChan <- fmt.Errorf("读取流失败: %w", err)
				return
			}

			// 去掉换行符
			line = strings.TrimSpace(line)

			// 跳过空行
			if line == "" {
				continue
			}

			// 检查是否是结束标记
			if line == "data: [DONE]" {
				break
			}

			// 解析 SSE 数据行
			if strings.HasPrefix(line, "data: ") {
				jsonStr := strings.TrimPrefix(line, "data: ")

				var chunk model.StreamChunk
				if err := json.Unmarshal([]byte(jsonStr), &chunk); err != nil {
					// 某些 API 可能发送非标准数据，跳过
					continue
				}

				// 提取内容并发送
				if len(chunk.Choices) > 0 {
					delta := chunk.Choices[0].Delta
					if delta.ReasoningContent != "" {
						contentChan <- model.StreamEvent{IsReasoning: true, Content: delta.ReasoningContent}
					}
					if delta.Content != "" {
						contentChan <- model.StreamEvent{IsReasoning: false, Content: delta.Content}
					}
				}
			}
		}
	}()

	return contentChan, errChan
}

// ============================================
// 技术讲解：Go Channel 和并发
// ============================================
//
// Channel 是 Go 语言的核心特性，用于 goroutine 之间的通信
//
// 创建 channel:
//   ch := make(chan string)      // 无缓冲 channel
//   ch := make(chan string, 10)  // 带缓冲 channel（容量 10）
//
// 发送数据:
//   ch <- "hello"
//
// 接收数据:
//   msg := <-ch
//
// 关闭 channel:
//   close(ch)  // 关闭后不能再发送，但可以继续接收剩余数据
//
// 在流式响应中的应用：
// 1. 主 goroutine 发起请求
// 2. 子 goroutine 读取 SSE 流，通过 channel 发送数据块
// 3. 调用方从 channel 接收数据，实时处理
//
// 这种模式的好处：
// - 解耦：生产者和消费者独立运行
// - 异步：不阻塞主线程
// - 安全：channel 是并发安全的

// ============================================
// ChatWithMultimodal 多模态对话（支持图片，非流式）
// ============================================
func (s *OpenAIService) ChatWithMultimodal(messages []model.OpenAIMultimodalMessage) (string, error) {
	// 构建请求体
	reqBody := map[string]interface{}{
		"model":       s.model,
		"messages":    messages,
		"temperature": 0.7,
		"stream":      false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	// 发送请求
	url := s.baseURL + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API 返回错误状态码 %d: %s", resp.StatusCode, string(body))
	}

	var chatResp model.OpenAIChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("AI API 错误: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("AI 没有返回任何回复")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// ============================================
// ChatStreamMultimodal 多模态流式对话（支持图片）
// ============================================
func (s *OpenAIService) ChatStreamMultimodal(messages []model.OpenAIMultimodalMessage) (<-chan model.StreamEvent, <-chan error) {
	contentChan := make(chan model.StreamEvent, 100)
	errChan := make(chan error, 1)

	go func() {
		defer close(contentChan)
		defer close(errChan)

		// 构建请求体
		reqBody := map[string]interface{}{
			"model":       s.model,
			"messages":    messages,
			"temperature": 0.7,
			"stream":      true,
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			errChan <- fmt.Errorf("序列化请求失败: %w", err)
			return
		}

		// 发送请求
		start := time.Now()
		url := s.baseURL + "/chat/completions"
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			errChan <- fmt.Errorf("创建请求失败: %w", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
		req.Header.Set("Accept", "text/event-stream")

		resp, err := s.httpClient.Do(req)
		if err != nil {
			errChan <- fmt.Errorf("发送请求失败: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errChan <- fmt.Errorf("API 返回错误状态码 %d: %s", resp.StatusCode, string(body))
			return
		}

		log.Printf("chat multimodal stream: upstream ok, status=%d, elapsed=%s", resp.StatusCode, time.Since(start))

		// 逐行读取 SSE 流
		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				errChan <- fmt.Errorf("读取流失败: %w", err)
				return
			}

			line = strings.TrimSpace(line)

			if line == "" {
				continue
			}

			if line == "data: [DONE]" {
				break
			}

			if strings.HasPrefix(line, "data: ") {
				jsonStr := strings.TrimPrefix(line, "data: ")

				var chunk model.StreamChunk
				if err := json.Unmarshal([]byte(jsonStr), &chunk); err != nil {
					continue
				}

				if len(chunk.Choices) > 0 {
					delta := chunk.Choices[0].Delta
					if delta.ReasoningContent != "" {
						contentChan <- model.StreamEvent{IsReasoning: true, Content: delta.ReasoningContent}
					}
					if delta.Content != "" {
						contentChan <- model.StreamEvent{IsReasoning: false, Content: delta.Content}
					}
				}
			}
		}
	}()

	return contentChan, errChan
}
