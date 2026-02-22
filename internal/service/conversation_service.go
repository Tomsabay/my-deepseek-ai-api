// Package service 包含对话相关的业务逻辑
package service

import (
	"errors"

	"gorm.io/gorm"

	"ai-api/internal/database"
	"ai-api/internal/model"
)

// ============================================
// 对话相关错误
// ============================================
var (
	ErrConversationNotFound = errors.New("对话不存在")
	ErrUnauthorized         = errors.New("无权访问此对话")
)

// ============================================
// ConversationService 对话服务
// ============================================
type ConversationService struct{}

// NewConversationService 创建对话服务实例
func NewConversationService() *ConversationService {
	return &ConversationService{}
}

// ============================================
// CreateConversation 创建新对话
// ============================================
func (s *ConversationService) CreateConversation(userID uint, title string) (*model.Conversation, error) {
	conversation := &model.Conversation{
		UserID: userID,
		Title:  title,
	}

	if err := database.DB.Create(conversation).Error; err != nil {
		return nil, err
	}

	return conversation, nil
}

// ============================================
// GetConversation 获取单个对话（包含消息）
// ============================================
func (s *ConversationService) GetConversation(conversationID, userID uint) (*model.Conversation, error) {
	var conversation model.Conversation

	// 预加载消息，按时间排序
	result := database.DB.Preload("Messages", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at ASC")
	}).First(&conversation, conversationID)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ErrConversationNotFound
	}
	if result.Error != nil {
		return nil, result.Error
	}

	// 检查权限
	if conversation.UserID != userID {
		return nil, ErrUnauthorized
	}

	return &conversation, nil
}

// ============================================
// ListConversations 获取用户的所有对话
// ============================================
func (s *ConversationService) ListConversations(userID uint) ([]model.Conversation, error) {
	var conversations []model.Conversation

	result := database.DB.Where("user_id = ?", userID).
		Order("updated_at DESC").
		Find(&conversations)

	return conversations, result.Error
}

// ============================================
// UpdateConversationTitle 更新对话标题
// ============================================
func (s *ConversationService) UpdateConversationTitle(conversationID, userID uint, title string) error {
	// 先检查权限
	var conversation model.Conversation
	result := database.DB.First(&conversation, conversationID)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return ErrConversationNotFound
	}
	if conversation.UserID != userID {
		return ErrUnauthorized
	}

	return database.DB.Model(&conversation).Update("title", title).Error
}

// ============================================
// DeleteConversation 删除对话
// ============================================
func (s *ConversationService) DeleteConversation(conversationID, userID uint) error {
	// 先检查权限
	var conversation model.Conversation
	result := database.DB.First(&conversation, conversationID)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return ErrConversationNotFound
	}
	if conversation.UserID != userID {
		return ErrUnauthorized
	}

	// 软删除对话及其消息
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// 删除消息
		if err := tx.Where("conversation_id = ?", conversationID).Delete(&model.Message{}).Error; err != nil {
			return err
		}
		// 删除对话
		return tx.Delete(&conversation).Error
	})
}

// ============================================
// AddMessage 添加消息到对话（带归属权校验）
// ============================================
func (s *ConversationService) AddMessage(conversationID, userID uint, role, content string) (*model.Message, error) {
	// 只有当 userID > 0 时才做归属权校验（兼容匿名模式）
	if userID > 0 {
		var conv model.Conversation
		if err := database.DB.First(&conv, conversationID).Error; err != nil {
			return nil, ErrConversationNotFound
		}
		if conv.UserID != userID {
			return nil, ErrUnauthorized
		}
	}

	message := &model.Message{
		ConversationID: conversationID,
		Role:           role,
		Content:        content,
	}

	if err := database.DB.Create(message).Error; err != nil {
		return nil, err
	}

	// 更新对话的 updated_at
	database.DB.Model(&model.Conversation{}).Where("id = ?", conversationID).Update("updated_at", message.CreatedAt)

	return message, nil
}

// ============================================
// GetMessages 获取对话的所有消息
// ============================================
func (s *ConversationService) GetMessages(conversationID, userID uint) ([]model.Message, error) {
	// 先检查对话权限
	var conversation model.Conversation
	result := database.DB.First(&conversation, conversationID)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ErrConversationNotFound
	}
	if conversation.UserID != userID {
		return nil, ErrUnauthorized
	}

	// 获取消息
	var messages []model.Message
	result = database.DB.Where("conversation_id = ?", conversationID).
		Order("created_at ASC").
		Find(&messages)

	return messages, result.Error
}

// ============================================
// ConvertToOpenAIMessages 转换为 OpenAI 消息格式
// ============================================
// 将数据库中的消息转换为 AI API 需要的格式
func (s *ConversationService) ConvertToOpenAIMessages(messages []model.Message) []model.OpenAIMessage {
	result := make([]model.OpenAIMessage, len(messages))
	for i, msg := range messages {
		result[i] = model.OpenAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return result
}
