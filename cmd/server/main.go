// Package main 是程序的入口包
// AI API 服务器启动文件 - 支持用户认证、对话持久化、CORS
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"ai-api/internal/api"
	"ai-api/internal/database"
	"ai-api/internal/middleware"
	"ai-api/internal/service"
)

// ============================================
// 配置常量和默认值
// ============================================
const (
	defaultPort         = "8080"
	defaultModel        = "gpt-3.5-turbo"
	defaultSystemPrompt = "你是一个友好、有帮助的 AI 助手。请用简洁清晰的中文回答用户的问题。"
	defaultDBPath       = "./data/ai-api.db"
	defaultJWTSecret    = "your-super-secret-jwt-key-change-in-production"
	defaultUploadDir    = "./data/uploads" // 上传文件存储目录
)

// ============================================
// main 程序入口
// ============================================
func main() {
	// ========== 第一步：加载配置 ==========
	config := loadConfig()

	// 验证必要的配置
	if config.apiKey == "" {
		log.Fatal("❌ 错误: 请设置 AI_API_KEY 环境变量")
	}

	// 打印配置信息
	log.Printf("📋 配置信息:")
	log.Printf("   - API 地址: %s", config.baseURL)
	log.Printf("   - 模型: %s", config.model)
	log.Printf("   - 端口: %s", config.port)
	log.Printf("   - 数据库: %s", config.dbPath)

	// ========== 第二步：初始化数据库 ==========
	// 确保数据目录存在
	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Fatalf("❌ 创建数据目录失败: %v", err)
	}

	if err := database.Init(config.dbPath); err != nil {
		log.Fatalf("❌ 数据库初始化失败: %v", err)
	}
	defer database.Close()

	// ========== 第三步：创建服务 ==========
	// AI 服务
	aiService := service.NewOpenAIService(
		config.baseURL,
		config.apiKey,
		config.model,
		config.systemPrompt,
	)

	// 认证服务
	authService := service.NewAuthService(config.jwtSecret)

	// 对话服务
	conversationService := service.NewConversationService()

	// 上传服务
	uploadService := service.NewUploadService(defaultUploadDir)

	// ========== 第四步：创建处理器 ==========
	chatHandler := api.NewHandler(aiService, conversationService, uploadService, authService)
	authHandler := api.NewAuthHandler(authService)
	conversationHandler := api.NewConversationHandler(conversationService, authService)
	uploadHandler := api.NewUploadHandler(uploadService)

	// ========== 第五步：设置 Gin 路由器 ==========
	r := gin.Default()

	// 全局中间件
	r.Use(middleware.CORSMiddleware()) // CORS 跨域支持

	// 托管前端静态文件
	r.StaticFile("/", "./web/index.html")
	r.Static("/static", "./web")

	// 托管上传文件（图片、文档等）
	r.Static("/uploads", defaultUploadDir)

	// 健康检查端点（公开）
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "欢迎使用 AI API 服务！",
			"status":  "running",
			"version": "2.0.0",
		})
	})

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// 注册各模块路由
	authHandler.RegisterRoutes(r)         // 认证路由（公开）
	chatHandler.RegisterRoutes(r)         // 聊天路由（支持匿名）
	conversationHandler.RegisterRoutes(r) // 对话管理路由（需认证）
	uploadHandler.RegisterRoutes(r)       // 文件上传路由

	// ========== 第六步：启动服务器 ==========
	log.Println("🚀 AI API 服务器正在启动...")
	log.Printf("📡 访问地址: http://localhost:%s", config.port)
	log.Printf("🌐 聊天界面: http://localhost:%s", config.port)
	log.Printf("💬 聊天接口: POST http://localhost:%s/api/v1/chat", config.port)
	log.Printf("🔐 认证接口: POST http://localhost:%s/api/v1/auth/login", config.port)
	log.Printf("📝 对话接口: GET http://localhost:%s/api/v1/conversations", config.port)
	log.Printf("📎 上传接口: POST http://localhost:%s/api/v1/upload", config.port)

	if err := r.Run(":" + config.port); err != nil {
		log.Fatalf("❌ 服务器启动失败: %v", err)
	}
}

// ============================================
// Config 配置结构
// ============================================
type Config struct {
	baseURL      string // AI API 基础地址
	apiKey       string // API 密钥
	model        string // 使用的模型
	systemPrompt string // 系统提示词
	port         string // 服务器端口
	dbPath       string // 数据库文件路径
	jwtSecret    string // JWT 签名密钥
}

// ============================================
// loadConfig 从环境变量加载配置
// ============================================
func loadConfig() Config {
	return Config{
		baseURL:      getEnv("AI_API_URL", "https://api.openai.com/v1"),
		apiKey:       getEnv("AI_API_KEY", ""),
		model:        getEnv("AI_MODEL", defaultModel),
		systemPrompt: getEnv("AI_SYSTEM_PROMPT", defaultSystemPrompt),
		port:         getEnv("PORT", defaultPort),
		dbPath:       getEnv("DB_PATH", defaultDBPath),
		jwtSecret:    getEnv("JWT_SECRET", defaultJWTSecret),
	}
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
