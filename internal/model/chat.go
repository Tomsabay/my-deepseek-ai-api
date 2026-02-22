// Package model 定义了 API 的数据结构
// 这些结构体用于定义请求参数和响应格式
package model

// ============================================
// ChatRequest 聊天请求结构体（支持多轮对话）
// ============================================
type ChatRequest struct {
	// Message 用户发送的消息（单轮对话时使用）
	// 如果同时提供了 Messages，则忽略此字段
	Message string `json:"message,omitempty"`

	// Messages 完整的对话历史（多轮对话时使用）
	// 格式：[{"role": "user", "content": "你好"}, {"role": "assistant", "content": "你好！"}]
	Messages []OpenAIMessage `json:"messages,omitempty"`

	// Model 可选，指定使用的模型（如 gpt-3.5-turbo, deepseek-chat 等）
	Model string `json:"model,omitempty"`

	// Stream 是否使用流式响应
	// true: 使用 SSE 逐字返回
	// false: 等待完整响应后返回（默认）
	Stream bool `json:"stream,omitempty"`

	// ConversationID 关联的对话 ID（可选，如果提供则自动保存消息）
	ConversationID uint `json:"conversation_id,omitempty"`

	// Attachments 附件列表（图片、文档等）
	// 当附件中包含图片时，会构建多模态请求发送给支持 Vision 的模型
	Attachments []MessageAttachment `json:"attachments,omitempty"`
}

// ChatResponse 聊天响应结构体（非流式）
type ChatResponse struct {
	Reply string `json:"reply"`
	Error string `json:"error,omitempty"`
}

// ============================================
// 流式响应相关结构（SSE - Server-Sent Events）
// ============================================

// StreamChunk 流式响应的单个数据块
// 每个数据块包含 AI 生成的一小段文本
type StreamChunk struct {
	// ID 响应 ID
	ID string `json:"id"`

	// Object 对象类型，流式时为 "chat.completion.chunk"
	Object string `json:"object"`

	// Created 创建时间戳
	Created int64 `json:"created"`

	// Model 使用的模型
	Model string `json:"model"`

	// Choices 选项列表
	Choices []StreamChoice `json:"choices"`
}

// StreamChoice 流式响应中的选项
type StreamChoice struct {
	// Index 选项索引
	Index int `json:"index"`

	// Delta 增量内容（只包含新生成的部分）
	Delta StreamDelta `json:"delta"`

	// FinishReason 结束原因（最后一个 chunk 才有值）
	// "stop": 正常结束
	// "length": 达到长度限制
	// null: 还在生成中
	FinishReason *string `json:"finish_reason"`
}

// StreamDelta 增量内容
type StreamDelta struct {
	// Role 角色（只在第一个 chunk 中出现）
	Role string `json:"role,omitempty"`

	// Content 内容片段（每个 chunk 包含一小段文本）
	Content string `json:"content,omitempty"`

	// ReasoningContent 思考过程（DeepSeek R1 等推理模型专有字段）
	ReasoningContent string `json:"reasoning_content,omitempty"`
}

// StreamEvent 流式事件，区分思考过程与正式回答
type StreamEvent struct {
	IsReasoning bool   // true 表示这是思考过程，false 表示正式回答
	Content     string // 内容文本
}

// ============================================
// OpenAI 兼容格式的数据结构
// 这是业界通用的 AI API 协议，很多大模型都支持
// ============================================

// OpenAIMessage 单条消息结构
// 对话是由多条消息组成的，每条消息有角色和内容
type OpenAIMessage struct {
	// Role 角色：system（系统设定）、user（用户）、assistant（AI助手）
	Role string `json:"role"`

	// Content 消息内容
	Content string `json:"content"`
}

// OpenAIChatRequest OpenAI Chat Completions API 请求格式
// 文档：https://platform.openai.com/docs/api-reference/chat/create
type OpenAIChatRequest struct {
	// Model 使用的模型名称，如 "gpt-3.5-turbo", "gpt-4"
	Model string `json:"model"`

	// Messages 对话消息列表
	Messages []OpenAIMessage `json:"messages"`

	// Temperature 温度参数，控制回复的随机性（0-2，越高越随机）
	Temperature float64 `json:"temperature,omitempty"`

	// MaxTokens 最大生成的 token 数量
	MaxTokens int `json:"max_tokens,omitempty"`

	// Stream 是否使用流式响应（SSE）
	Stream bool `json:"stream,omitempty"`
}

// OpenAIChatResponse OpenAI Chat Completions API 响应格式
type OpenAIChatResponse struct {
	// ID 响应的唯一标识符
	ID string `json:"id"`

	// Object 对象类型，通常是 "chat.completion"
	Object string `json:"object"`

	// Created 创建时间（Unix 时间戳）
	Created int64 `json:"created"`

	// Model 使用的模型
	Model string `json:"model"`

	// Choices AI 生成的回复选项（通常只有一个）
	Choices []OpenAIChoice `json:"choices"`

	// Usage Token 使用统计
	Usage OpenAIUsage `json:"usage,omitempty"`

	// Error 错误信息（如果有）
	Error *OpenAIError `json:"error,omitempty"`
}

// OpenAIChoice 单个回复选项
type OpenAIChoice struct {
	// Index 选项索引
	Index int `json:"index"`

	// Message AI 回复的消息
	Message OpenAIMessage `json:"message"`

	// FinishReason 结束原因：stop（正常结束）、length（达到长度限制）等
	FinishReason string `json:"finish_reason"`
}

// OpenAIUsage Token 使用统计
type OpenAIUsage struct {
	// PromptTokens 提示词使用的 token 数
	PromptTokens int `json:"prompt_tokens"`

	// CompletionTokens 回复使用的 token 数
	CompletionTokens int `json:"completion_tokens"`

	// TotalTokens 总 token 数
	TotalTokens int `json:"total_tokens"`
}

// OpenAIError OpenAI API 错误格式
type OpenAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// ============================================
// 技术讲解：多轮对话和流式响应
// ============================================
//
// 【多轮对话原理】
// AI 本身不记忆对话历史，每次请求都是独立的。
// 为了实现"记忆"，我们需要在每次请求时把完整的对话历史发送给 AI：
//
// 请求1: messages = [{user: "你好"}]
// 响应1: "你好！"
//
// 请求2: messages = [{user: "你好"}, {assistant: "你好！"}, {user: "我叫小明"}]
// 响应2: "你好小明！"
//
// 请求3: messages = [... 之前所有消息 ..., {user: "还记得我叫什么吗？"}]
// 响应3: "你叫小明！"
//
// 【流式响应原理】
// 普通响应：AI 生成完所有内容后，一次性返回
// 流式响应：AI 边生成边返回，每生成一个 token 就推送一次
//
// 使用 Server-Sent Events (SSE) 协议：
// - Content-Type: text/event-stream
// - 每个事件格式：data: {json}\n\n
// - 结束标记：data: [DONE]\n\n
//
// 示例流式数据：
// data: {"choices":[{"delta":{"content":"你"}}]}
// data: {"choices":[{"delta":{"content":"好"}}]}
// data: {"choices":[{"delta":{"content":"！"}}]}
// data: [DONE]
