// Package middleware 包含 HTTP 中间件
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"ai-api/internal/service"
)

// ============================================
// AuthMiddleware JWT 认证中间件
// ============================================
// 验证请求中的 JWT Token，并将用户 ID 存入 Context
//
// 使用方式：
//
//	r.Use(middleware.AuthMiddleware(authService))
//
// 或者只对特定路由组应用：
//
//	protected := r.Group("/api/v1")
//	protected.Use(middleware.AuthMiddleware(authService))
//
// 在 Handler 中获取用户 ID：
//
//	userID, _ := c.Get("userID")
func AuthMiddleware(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取 Token
		// 格式：Authorization: Bearer <token>
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "未提供认证 Token",
			})
			c.Abort()
			return
		}

		// 解析 Bearer Token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Token 格式错误，应为 Bearer <token>",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 验证 Token
		userID, err := authService.ValidateToken(tokenString)
		if err != nil {
			status := http.StatusUnauthorized
			message := "Token 验证失败"

			switch err {
			case service.ErrTokenExpired:
				message = "Token 已过期，请重新登录"
			case service.ErrInvalidToken:
				message = "无效的 Token"
			}

			c.JSON(status, gin.H{
				"error": message,
			})
			c.Abort()
			return
		}

		// 将用户 ID 存入 Context，供后续 Handler 使用
		c.Set("userID", userID)

		// 继续处理请求
		c.Next()
	}
}

// ============================================
// OptionalAuthMiddleware 可选认证中间件
// ============================================
// 如果提供了 Token 就验证，没有也不阻止请求
// 适用于既支持匿名访问又支持登录用户的接口
func OptionalAuthMiddleware(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		userID, err := authService.ValidateToken(parts[1])
		if err == nil {
			c.Set("userID", userID)
		}

		c.Next()
	}
}

// ============================================
// 技术讲解：Gin 中间件
// ============================================
//
// 中间件是在请求处理前后执行的函数，用于：
// - 认证/授权检查
// - 日志记录
// - 错误处理
// - 请求限流
// - CORS 处理
//
// 中间件执行流程：
//
//	请求 → 中间件1 → 中间件2 → Handler → 中间件2 → 中间件1 → 响应
//	       (before)  (before)  (处理)    (after)   (after)
//
// Gin 中间件示例：
//
//	func MyMiddleware() gin.HandlerFunc {
//	    return func(c *gin.Context) {
//	        // 请求处理前的逻辑
//	        start := time.Now()
//
//	        c.Next()  // 调用下一个 Handler
//
//	        // 请求处理后的逻辑
//	        latency := time.Since(start)
//	        log.Printf("请求耗时: %v", latency)
//	    }
//	}
//
// c.Abort() 会终止后续 Handler 的执行
// c.Set(key, value) 可以在 Context 中存储数据
// c.Get(key) 可以获取存储的数据
