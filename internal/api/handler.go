// Package api 包含 HTTP API 的处理逻辑
// 这里定义了路由和请求处理器
package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"ai-api/internal/middleware"
	"ai-api/internal/model"
	"ai-api/internal/service"
)

// ============================================
// Handler API 处理器
// ============================================
type Handler struct {
	aiService           service.AIService
	conversationService *service.ConversationService
	uploadService       *service.UploadService
	authService         *service.AuthService // 用于 Chat 路由的可选认证
}

// ============================================
// NewHandler 创建一个新的 Handler 实例
// ============================================
func NewHandler(aiService service.AIService, conversationService *service.ConversationService, uploadService *service.UploadService, authService *service.AuthService) *Handler {
	return &Handler{
		aiService:           aiService,
		conversationService: conversationService,
		uploadService:       uploadService,
		authService:         authService,
	}
}

// ============================================
// RegisterRoutes 注册所有 API 路由
// ============================================
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	// 聊天路由使用可选认证：登录用户绑定对话，未登录也能用基础聊天
	v1.Use(middleware.OptionalAuthMiddleware(h.authService))
	{
		// POST /api/v1/chat - 聊天接口（支持多轮对话和流式响应）
		v1.POST("/chat", h.Chat)

		// POST /api/v1/chat/stream - 专用流式聊天接口
		v1.POST("/chat/stream", h.ChatStream)
	}
}

// ============================================
// Chat 处理聊天请求（支持多轮对话、流式响应和图片附件）
// ============================================
func (h *Handler) Chat(c *gin.Context) {
	var req model.ChatRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ChatResponse{
			Error: "请求参数错误: " + err.Error(),
		})
		return
	}

	// 获取当前登录用户 ID（可选认证，可能为 0）
	var currentUserID uint
	if uid, exists := c.Get("userID"); exists {
		currentUserID = uid.(uint)
	}

	// 尝试保存用户消息（如果有 ConversationID）
	// 必须是登录用户且是对话拥有者才能写入，防止跨账号串数据
	if req.ConversationID > 0 {
		if currentUserID == 0 {
			// 未登录用户不能往指定对话写入消息
			c.JSON(http.StatusUnauthorized, model.ChatResponse{
				Error: "请先登录再使用对话功能",
			})
			return
		}
		// 校验对话归属权：只有对话拥有者才能写入消息
		_, err := h.conversationService.GetConversation(req.ConversationID, currentUserID)
		if err != nil {
			log.Printf("chat: 用户 %d 尝试访问不属于自己的对话 %d", currentUserID, req.ConversationID)
			c.JSON(http.StatusForbidden, model.ChatResponse{
				Error: "无权访问此对话",
			})
			return
		}

		userContent := req.Message
		if len(req.Messages) > 0 {
			lastMsg := req.Messages[len(req.Messages)-1]
			if lastMsg.Role == "user" {
				userContent = lastMsg.Content
			}
		}

		if userContent != "" {
			h.conversationService.AddMessage(req.ConversationID, currentUserID, "user", userContent)
		}
	}

	// 构建消息列表
	messages := h.buildMessages(req)

	// 动态模型选择：如果请求中指定了 model，则尝试使用该模型
	// 这里用局部变量择而非直接操作字段，避免影响其他请求
	aiSvc := h.aiService
	if req.Model != "" {
		// 类型断言：将接口转换为具体的 *service.OpenAIService
		if openAISvc, ok := h.aiService.(*service.OpenAIService); ok {
			// WithModel 返回一个仅修改了 model 字段的克隆体，不会影响全局服务
			aiSvc = openAISvc.WithModel(req.Model)
			log.Printf("chat: 使用用户指定模型: %s", req.Model)
		}
	}

	// 检查是否有图片附件，如果有则需要特殊处理
	hasImages := false
	for _, att := range req.Attachments {
		if att.Type == "image" {
			hasImages = true
			break
		}
	}

	// 如果有图片附件，构建多模态消息
	if hasImages && h.uploadService != nil {
		multimodalMessages := h.buildMultimodalMessages(messages, req.Attachments)
		log.Printf("chat: multimodal request with %d images, conversation_id=%d", len(req.Attachments), req.ConversationID)

		// 检查是否需要流式响应
		if req.Stream {
			h.handleMultimodalStreamResponse(c, multimodalMessages, req.ConversationID, currentUserID, aiSvc)
			return
		}

		// 非流式多模态请求
		reply, err := aiSvc.ChatWithMultimodal(multimodalMessages)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.ChatResponse{
				Error: "AI 服务错误: " + err.Error(),
			})
			return
		}

		// 保存 AI 回复
		if req.ConversationID > 0 {
			h.conversationService.AddMessage(req.ConversationID, currentUserID, "assistant", reply)
		}

		c.JSON(http.StatusOK, model.ChatResponse{
			Reply: reply,
		})
		return
	}

	// 检查是否需要流式响应（无图片的普通请求）
	if req.Stream {
		log.Printf("chat: stream request, conversation_id=%d, messages=%d", req.ConversationID, len(messages))
		h.handleStreamResponse(c, messages, req.ConversationID, currentUserID, aiSvc)
		return
	}

	// 非流式：调用多轮对话方法
	log.Printf("chat: request, conversation_id=%d, messages=%d", req.ConversationID, len(messages))
	reply, err := aiSvc.ChatWithHistory(messages)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ChatResponse{
			Error: "AI 服务错误: " + err.Error(),
		})
		return
	}

	// 保存 AI 回复（如果有 ConversationID 且已登录）
	if req.ConversationID > 0 && currentUserID > 0 {
		h.conversationService.AddMessage(req.ConversationID, currentUserID, "assistant", reply)
	}

	c.JSON(http.StatusOK, model.ChatResponse{
		Reply: reply,
	})
}

// ============================================
// ChatStream 专用流式聊天接口
// ============================================
func (h *Handler) ChatStream(c *gin.Context) {
	var req model.ChatRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ChatResponse{
			Error: "请求参数错误: " + err.Error(),
		})
		return
	}

	// 获取当前用户 ID（可选认证）
	var currentUserID uint
	if uid, exists := c.Get("userID"); exists {
		currentUserID = uid.(uint)
	}

	// 校验对话归属权
	if req.ConversationID > 0 && currentUserID > 0 {
		_, err := h.conversationService.GetConversation(req.ConversationID, currentUserID)
		if err != nil {
			log.Printf("chat/stream: 用户 %d 尝试访问不属于自己的对话 %d", currentUserID, req.ConversationID)
			c.JSON(http.StatusForbidden, model.ChatResponse{
				Error: "无权访问此对话",
			})
			return
		}
	}

	messages := h.buildMessages(req)

	// 同样支持动态模型选择
	aiSvc := h.aiService
	if req.Model != "" {
		if openAISvc, ok := h.aiService.(*service.OpenAIService); ok {
			aiSvc = openAISvc.WithModel(req.Model)
		}
	}

	h.handleStreamResponse(c, messages, req.ConversationID, currentUserID, aiSvc)
}

// ============================================
// buildMessages 构建消息列表
// ============================================
func (h *Handler) buildMessages(req model.ChatRequest) []model.OpenAIMessage {
	if len(req.Messages) > 0 {
		if req.Messages[0].Role != "system" {
			messages := make([]model.OpenAIMessage, 0, len(req.Messages)+1)
			messages = append(messages, model.OpenAIMessage{
				Role:    "system",
				Content: h.aiService.GetSystemPrompt(),
			})
			messages = append(messages, req.Messages...)
			return messages
		}
		return req.Messages
	}

	return []model.OpenAIMessage{
		{Role: "system", Content: h.aiService.GetSystemPrompt()},
		{Role: "user", Content: req.Message},
	}
}

// ============================================
// buildMultimodalMessages 构建多模态消息（包含图片）
// ============================================
func (h *Handler) buildMultimodalMessages(messages []model.OpenAIMessage, attachments []model.MessageAttachment) []model.OpenAIMultimodalMessage {
	result := make([]model.OpenAIMultimodalMessage, 0, len(messages))

	for i, msg := range messages {
		// 只有最后一条用户消息需要附加图片
		if i == len(messages)-1 && msg.Role == "user" {
			// 构建多模态内容数组
			contents := []model.MultimodalContent{
				{
					Type: "text",
					Text: msg.Content,
				},
			}

			// 添加图片
			for _, att := range attachments {
				if att.Type == "image" {
					// 将图片转换为 base64 data URL
					dataURL, err := h.uploadService.BuildImageDataURL(att.URL)
					if err != nil {
						log.Printf("警告：无法读取图片 %s: %v", att.URL, err)
						continue
					}

					contents = append(contents, model.MultimodalContent{
						Type: "image_url",
						ImageURL: &model.ImageURLContent{
							URL:    dataURL,
							Detail: "auto",
						},
					})
				}
			}

			result = append(result, model.OpenAIMultimodalMessage{
				Role:    msg.Role,
				Content: contents,
			})
		} else {
			// 普通文本消息
			result = append(result, model.OpenAIMultimodalMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	return result
}

// ============================================
// handleStreamResponse 处理流式响应
// aiSvc 支持动态模型切换，如果请求指定了模型，就用克隆后的实例
func (h *Handler) handleStreamResponse(c *gin.Context, messages []model.OpenAIMessage, conversationID, userID uint, aiSvc service.AIService) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	c.Header("X-Accel-Buffering", "no") // 关键！强制关闭代理缓存缓冲以实现实时流输出
	c.Header("Access-Control-Allow-Origin", "*")

	w := c.Writer
	fullReply := ""

	log.Printf("chat: stream start, conversation_id=%d, messages=%d", conversationID, len(messages))
	// 调用传入的 aiSvc（可能是克隆的不同模型实例）
	contentChan, errChan := aiSvc.ChatStream(messages)

	for {
		select {
		case event, ok := <-contentChan:
			if !ok {
				fmt.Fprintf(w, "data: [DONE]\n\n")
				w.Flush()

				if conversationID > 0 && userID > 0 && fullReply != "" {
					h.conversationService.AddMessage(conversationID, userID, "assistant", fullReply)
				}
				log.Printf("chat: stream done, conversation_id=%d, size=%d", conversationID, len(fullReply))
				return
			}

			if event.IsReasoning {
				fmt.Fprintf(w, "data: {\"reasoning\":%q}\n\n", event.Content)
			} else {
				fullReply += event.Content
				fmt.Fprintf(w, "data: {\"content\":%q}\n\n", event.Content)
			}
			w.Flush()

		case err := <-errChan:
			if err != nil {
				fmt.Fprintf(w, "data: {\"error\":%q}\n\n", err.Error())
				w.Flush()
				log.Printf("chat: stream error, conversation_id=%d, err=%v", conversationID, err)
				return
			}
		}
	}
}

// ============================================
// handleMultimodalStreamResponse 处理多模态流式响应
func (h *Handler) handleMultimodalStreamResponse(c *gin.Context, messages []model.OpenAIMultimodalMessage, conversationID, userID uint, aiSvc service.AIService) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	c.Header("X-Accel-Buffering", "no") // 同上
	c.Header("Access-Control-Allow-Origin", "*")

	w := c.Writer
	fullReply := ""

	log.Printf("chat: multimodal stream start, conversation_id=%d, messages=%d", conversationID, len(messages))
	// 使用动态指定的 aiSvc 实例
	contentChan, errChan := aiSvc.ChatStreamMultimodal(messages)

	for {
		select {
		case event, ok := <-contentChan:
			if !ok {
				fmt.Fprintf(w, "data: [DONE]\n\n")
				w.Flush()

				if conversationID > 0 && userID > 0 && fullReply != "" {
					h.conversationService.AddMessage(conversationID, userID, "assistant", fullReply)
				}
				log.Printf("chat: multimodal stream done, conversation_id=%d, size=%d", conversationID, len(fullReply))
				return
			}

			if event.IsReasoning {
				fmt.Fprintf(w, "data: {\"reasoning\":%q}\n\n", event.Content)
			} else {
				fullReply += event.Content
				fmt.Fprintf(w, "data: {\"content\":%q}\n\n", event.Content)
			}
			w.Flush()

		case err := <-errChan:
			if err != nil {
				fmt.Fprintf(w, "data: {\"error\":%q}\n\n", err.Error())
				w.Flush()
				log.Printf("chat: multimodal stream error, conversation_id=%d, err=%v", conversationID, err)
				return
			}
		}
	}
}
