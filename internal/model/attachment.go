// Package model 定义了 API 的数据结构
// 这个文件定义了附件（图片、文件）相关的数据结构
package model

import "time"

// ============================================
// Attachment 附件结构体
// ============================================
// 用于存储上传的图片和文件信息
type Attachment struct {
	// ID 附件唯一标识
	ID string `json:"id"`

	// Filename 原始文件名
	Filename string `json:"filename"`

	// Type 附件类型：image（图片）、document（文档）、file（其他文件）
	Type string `json:"type"`

	// MimeType MIME 类型，如 image/png, application/pdf
	MimeType string `json:"mime_type"`

	// Size 文件大小（字节）
	Size int64 `json:"size"`

	// URL 访问 URL（相对路径）
	URL string `json:"url"`

	// ThumbnailURL 缩略图 URL（仅图片类型有）
	ThumbnailURL string `json:"thumbnail_url,omitempty"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`
}

// ============================================
// UploadResponse 上传响应
// ============================================
type UploadResponse struct {
	// Success 是否成功
	Success bool `json:"success"`

	// Attachment 上传的附件信息
	Attachment *Attachment `json:"attachment,omitempty"`

	// Error 错误信息
	Error string `json:"error,omitempty"`
}

// ============================================
// MessageAttachment 消息中的附件引用
// ============================================
// 用于在聊天请求中引用附件
type MessageAttachment struct {
	// ID 附件 ID
	ID string `json:"id"`

	// Type 附件类型
	Type string `json:"type"`

	// URL 附件 URL
	URL string `json:"url"`

	// Filename 文件名
	Filename string `json:"filename"`
}

// ============================================
// OpenAI Vision API 支持的多模态消息格式
// ============================================

// MultimodalContent 多模态内容（支持文本和图片混合）
type MultimodalContent struct {
	// Type 内容类型：text 或 image_url
	Type string `json:"type"`

	// Text 文本内容（当 Type 为 text 时使用）
	Text string `json:"text,omitempty"`

	// ImageURL 图片 URL（当 Type 为 image_url 时使用）
	ImageURL *ImageURLContent `json:"image_url,omitempty"`
}

// ImageURLContent 图片 URL 内容
type ImageURLContent struct {
	// URL 图片的 URL 或 base64 数据
	// 格式：https://... 或 data:image/jpeg;base64,...
	URL string `json:"url"`

	// Detail 图片细节级别：low, high, auto
	Detail string `json:"detail,omitempty"`
}

// OpenAIMultimodalMessage 支持多模态的消息结构
// 用于发送包含图片的请求到支持 Vision 的模型
type OpenAIMultimodalMessage struct {
	// Role 角色：system、user、assistant
	Role string `json:"role"`

	// Content 内容（可以是字符串或多模态内容数组）
	// 当只有文本时，可以直接用字符串
	// 当包含图片时，必须用 MultimodalContent 数组
	Content interface{} `json:"content"`
}

// ============================================
// 技术讲解：多模态 AI 接口
// ============================================
//
// 【什么是多模态】
// 多模态（Multimodal）指 AI 能够处理多种类型的输入，
// 如文本 + 图片、文本 + 音频等。
//
// 【OpenAI Vision API 格式】
// 普通文本消息：
// {
//     "role": "user",
//     "content": "你好"
// }
//
// 包含图片的多模态消息：
// {
//     "role": "user",
//     "content": [
//         {"type": "text", "text": "请描述这张图片"},
//         {"type": "image_url", "image_url": {"url": "data:image/jpeg;base64,..."}}
//     ]
// }
//
// 【支持多模态的模型】
// - OpenAI: gpt-4-vision-preview, gpt-4o
// - DeepSeek: deepseek-chat (支持 Vision)
// - 通义千问: qwen-vl-plus
