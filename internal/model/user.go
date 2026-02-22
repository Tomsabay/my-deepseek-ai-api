// Package model 定义了数据库模型
// 包含用户、对话、消息等实体
package model

import (
	"time"

	"gorm.io/gorm"
)

// ============================================
// User 用户模型
// ============================================
// 存储用户的基本信息和认证数据
type User struct {
	// ID 用户唯一标识（自增主键）
	ID uint `gorm:"primaryKey" json:"id"`

	// Username 用户名（唯一，用于登录）
	Username string `gorm:"uniqueIndex;size:50;not null" json:"username"`

	// PasswordHash 密码哈希值（使用 bcrypt 加密）
	// 注意：不要在 JSON 中暴露密码！
	PasswordHash string `gorm:"size:255;not null" json:"-"`

	// Nickname 昵称（显示名称）
	Nickname string `gorm:"size:50" json:"nickname"`

	// Avatar 头像 URL
	Avatar string `gorm:"size:255" json:"avatar,omitempty"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at"`

	// DeletedAt 软删除时间（GORM 软删除支持）
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Conversations 用户的对话列表（一对多关系）
	Conversations []Conversation `gorm:"foreignKey:UserID" json:"-"`
}

// ============================================
// Conversation 对话模型
// ============================================
// 每个对话包含多条消息，属于一个用户
type Conversation struct {
	// ID 对话唯一标识
	ID uint `gorm:"primaryKey" json:"id"`

	// UserID 所属用户 ID（外键）
	UserID uint `gorm:"index;not null" json:"user_id"`

	// Title 对话标题（可选，用于显示）
	Title string `gorm:"size:200" json:"title"`

	// Model 使用的 AI 模型
	Model string `gorm:"size:50" json:"model,omitempty"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at"`

	// DeletedAt 软删除时间
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Messages 对话中的消息列表（一对多关系）
	Messages []Message `gorm:"foreignKey:ConversationID" json:"messages,omitempty"`
}

// ============================================
// Message 消息模型
// ============================================
// 对话中的每条消息
type Message struct {
	// ID 消息唯一标识
	ID uint `gorm:"primaryKey" json:"id"`

	// ConversationID 所属对话 ID（外键）
	ConversationID uint `gorm:"index;not null" json:"conversation_id"`

	// Role 角色：user（用户）、assistant（AI助手）、system（系统）
	Role string `gorm:"size:20;not null" json:"role"`

	// Content 消息内容
	Content string `gorm:"type:text;not null" json:"content"`

	// TokenCount Token 数量（可选，用于统计）
	TokenCount int `json:"token_count,omitempty"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`
}

// ============================================
// 技术讲解：GORM 模型定义
// ============================================
//
// GORM 是 Go 语言最流行的 ORM（对象关系映射）框架
// 它让我们可以用 Go 结构体来操作数据库，而不是写 SQL
//
// 结构体标签说明：
// - gorm:"primaryKey"       设置为主键
// - gorm:"uniqueIndex"      唯一索引
// - gorm:"index"            普通索引
// - gorm:"size:100"         字段长度
// - gorm:"not null"         非空约束
// - gorm:"type:text"        指定数据库类型
// - gorm:"foreignKey:XXX"   外键关系
//
// JSON 标签说明：
// - json:"name"             JSON 字段名
// - json:"-"                不包含在 JSON 中
// - json:"name,omitempty"   空值时省略
//
// 关系说明：
// - User 1:N Conversation   一个用户有多个对话
// - Conversation 1:N Message 一个对话有多条消息
