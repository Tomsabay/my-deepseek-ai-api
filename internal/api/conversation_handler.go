// Package api 包含对话管理相关的 API 处理器
package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ai-api/internal/middleware"
	"ai-api/internal/service"
)

// ============================================
// ConversationHandler 对话处理器
// ============================================
type ConversationHandler struct {
	conversationService *service.ConversationService
	authService         *service.AuthService
}

// NewConversationHandler 创建对话处理器
func NewConversationHandler(convService *service.ConversationService, authService *service.AuthService) *ConversationHandler {
	return &ConversationHandler{
		conversationService: convService,
		authService:         authService,
	}
}

// ============================================
// 请求/响应结构体
// ============================================

// CreateConversationRequest 创建对话请求
type CreateConversationRequest struct {
	Title string `json:"title,omitempty"`
}

// UpdateTitleRequest 更新标题请求
type UpdateTitleRequest struct {
	Title string `json:"title" binding:"required"`
}

// ============================================
// RegisterRoutes 注册对话相关路由
// ============================================
func (h *ConversationHandler) RegisterRoutes(r *gin.Engine) {
	// 需要认证的对话接口
	conversations := r.Group("/api/v1/conversations")
	conversations.Use(middleware.AuthMiddleware(h.authService))
	{
		// GET /api/v1/conversations - 获取对话列表
		conversations.GET("", h.ListConversations)

		// POST /api/v1/conversations - 创建新对话
		conversations.POST("", h.CreateConversation)

		// GET /api/v1/conversations/:id - 获取单个对话（包含消息）
		conversations.GET("/:id", h.GetConversation)

		// PUT /api/v1/conversations/:id - 更新对话标题
		conversations.PUT("/:id", h.UpdateConversation)

		// DELETE /api/v1/conversations/:id - 删除对话
		conversations.DELETE("/:id", h.DeleteConversation)

		// GET /api/v1/conversations/:id/messages - 获取对话消息
		conversations.GET("/:id/messages", h.GetMessages)
	}
}

// ============================================
// ListConversations 获取对话列表
// ============================================
// GET /api/v1/conversations
//
// 响应：
//
//	{
//	    "conversations": [
//	        {"id": 1, "title": "关于 Go 语言的讨论", "created_at": "..."},
//	        ...
//	    ]
//	}
func (h *ConversationHandler) ListConversations(c *gin.Context) {
	userID := h.getUserID(c)

	conversations, err := h.conversationService.ListConversations(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取对话列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"conversations": conversations,
	})
}

// ============================================
// CreateConversation 创建新对话
// ============================================
// POST /api/v1/conversations
//
// 请求体：
//
//	{
//	    "title": "新对话"
//	}
func (h *ConversationHandler) CreateConversation(c *gin.Context) {
	userID := h.getUserID(c)

	var req CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 没有请求体也可以创建对话
		req.Title = "新对话"
	}

	if req.Title == "" {
		req.Title = "新对话"
	}

	conversation, err := h.conversationService.CreateConversation(userID, req.Title)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "创建对话失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, conversation)
}

// ============================================
// GetConversation 获取单个对话
// ============================================
// GET /api/v1/conversations/:id
func (h *ConversationHandler) GetConversation(c *gin.Context) {
	userID := h.getUserID(c)
	conversationID, err := h.parseID(c, "id")
	if err != nil {
		return
	}

	conversation, err := h.conversationService.GetConversation(conversationID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		if err == service.ErrConversationNotFound {
			status = http.StatusNotFound
		} else if err == service.ErrUnauthorized {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, conversation)
}

// ============================================
// UpdateConversation 更新对话标题
// ============================================
// PUT /api/v1/conversations/:id
//
// 请求体：
//
//	{
//	    "title": "新标题"
//	}
func (h *ConversationHandler) UpdateConversation(c *gin.Context) {
	userID := h.getUserID(c)
	conversationID, err := h.parseID(c, "id")
	if err != nil {
		return
	}

	var req UpdateTitleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "请求参数错误: " + err.Error(),
		})
		return
	}

	err = h.conversationService.UpdateConversationTitle(conversationID, userID, req.Title)
	if err != nil {
		status := http.StatusInternalServerError
		if err == service.ErrConversationNotFound {
			status = http.StatusNotFound
		} else if err == service.ErrUnauthorized {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "更新成功",
	})
}

// ============================================
// DeleteConversation 删除对话
// ============================================
// DELETE /api/v1/conversations/:id
func (h *ConversationHandler) DeleteConversation(c *gin.Context) {
	userID := h.getUserID(c)
	conversationID, err := h.parseID(c, "id")
	if err != nil {
		return
	}

	err = h.conversationService.DeleteConversation(conversationID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		if err == service.ErrConversationNotFound {
			status = http.StatusNotFound
		} else if err == service.ErrUnauthorized {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "删除成功",
	})
}

// ============================================
// GetMessages 获取对话消息
// ============================================
// GET /api/v1/conversations/:id/messages
func (h *ConversationHandler) GetMessages(c *gin.Context) {
	userID := h.getUserID(c)
	conversationID, err := h.parseID(c, "id")
	if err != nil {
		return
	}

	messages, err := h.conversationService.GetMessages(conversationID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		if err == service.ErrConversationNotFound {
			status = http.StatusNotFound
		} else if err == service.ErrUnauthorized {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
	})
}

// ============================================
// 辅助方法
// ============================================

// getUserID 从 Context 获取用户 ID
func (h *ConversationHandler) getUserID(c *gin.Context) uint {
	userID, exists := c.Get("userID")
	if !exists {
		return 0
	}
	return userID.(uint)
}

// parseID 解析 URL 中的 ID 参数
func (h *ConversationHandler) parseID(c *gin.Context, param string) (uint, error) {
	idStr := c.Param(param)
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的 ID 参数",
		})
		return 0, err
	}
	return uint(id), nil
}
