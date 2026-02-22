// Package database 负责数据库连接和初始化
package database

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ai-api/internal/model"
)

// DB 全局数据库连接实例
var DB *gorm.DB

// ============================================
// Init 初始化数据库连接
// ============================================
// 参数：
//   - dbPath: 数据库文件路径（如 "data/app.db"）
//
// 返回：
//   - error: 初始化错误
//
// 技术讲解：
// SQLite 是一个嵌入式数据库，数据存储在单个文件中
// 优点：无需安装数据库服务器，适合开发和小型应用
// 缺点：不适合高并发写入场景
func Init(dbPath string) error {
	var err error

	// 配置 GORM
	config := &gorm.Config{
		// 设置日志级别（开发时用 Info，生产用 Warn）
		Logger: logger.Default.LogMode(logger.Info),
	}

	// 连接数据库
	DB, err = gorm.Open(sqlite.Open(dbPath), config)
	if err != nil {
		return err
	}

	log.Printf("📦 数据库连接成功: %s", dbPath)

	// 自动迁移（创建/更新表结构）
	// AutoMigrate 会根据模型自动创建表，添加缺失的列
	// 注意：它不会删除列或修改列类型
	err = DB.AutoMigrate(
		&model.User{},
		&model.Conversation{},
		&model.Message{},
	)
	if err != nil {
		return err
	}

	log.Println("📋 数据库表迁移完成")
	return nil
}

// ============================================
// Close 关闭数据库连接
// ============================================
func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// ============================================
// 技术讲解：GORM 基本操作
// ============================================
//
// 创建记录：
//   user := &model.User{Username: "test", PasswordHash: "xxx"}
//   DB.Create(user)                      // INSERT INTO users ...
//
// 查询记录：
//   var user model.User
//   DB.First(&user, 1)                   // SELECT * FROM users WHERE id = 1
//   DB.Where("username = ?", "test").First(&user)
//
// 更新记录：
//   DB.Model(&user).Update("nickname", "新昵称")
//   DB.Model(&user).Updates(map[string]interface{}{"nickname": "xxx"})
//
// 删除记录：
//   DB.Delete(&user, 1)                  // 软删除（设置 deleted_at）
//   DB.Unscoped().Delete(&user, 1)       // 硬删除（真正删除）
//
// 关联查询：
//   DB.Preload("Conversations").Find(&user)  // 预加载对话
//   DB.Preload("Messages").Find(&conversation)
