// Package middleware 包含 CORS 配置
package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// ============================================
// CORSMiddleware 跨域资源共享中间件
// ============================================
// 允许前端跨域访问 API
//
// 技术讲解：
// CORS (Cross-Origin Resource Sharing) 是浏览器的安全机制
// 当前端域名与后端 API 域名不同时，浏览器会阻止请求
// 通过设置响应头，可以允许特定来源的跨域请求
//
// 相关响应头：
// - Access-Control-Allow-Origin: 允许的来源
// - Access-Control-Allow-Methods: 允许的 HTTP 方法
// - Access-Control-Allow-Headers: 允许的请求头
// - Access-Control-Allow-Credentials: 是否允许携带 Cookie
// - Access-Control-Max-Age: 预检请求缓存时间
func CORSMiddleware() gin.HandlerFunc {
	config := cors.Config{
		// 允许所有来源（开发环境）
		// 生产环境建议设置为具体的域名列表
		// AllowOrigins: []string{"https://your-frontend.com"},
		AllowAllOrigins: true,

		// 允许的 HTTP 方法
		AllowMethods: []string{
			"GET",
			"POST",
			"PUT",
			"PATCH",
			"DELETE",
			"OPTIONS",
		},

		// 允许的请求头
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Content-Length",
			"Accept",
			"Accept-Encoding",
			"Authorization",
			"X-Requested-With",
		},

		// 暴露给前端的响应头
		ExposeHeaders: []string{
			"Content-Length",
			"Content-Type",
		},

		// 是否允许携带 Cookie
		AllowCredentials: true,

		// 预检请求缓存时间（12 小时）
		// 减少 OPTIONS 请求的次数
		MaxAge: 12 * time.Hour,
	}

	return cors.New(config)
}

// ============================================
// 技术讲解：CORS 预检请求
// ============================================
//
// 对于某些请求（如 POST JSON），浏览器会先发送 OPTIONS 请求
// 这个请求叫做"预检请求"（Preflight Request）
//
// 触发预检的条件：
// 1. 使用 PUT, DELETE, PATCH 方法
// 2. Content-Type 不是 application/x-www-form-urlencoded,
//    multipart/form-data, text/plain
// 3. 包含自定义请求头（如 Authorization）
//
// 预检请求流程：
//
//	1. 浏览器发送 OPTIONS 请求
//	   OPTIONS /api/v1/chat
//	   Origin: http://localhost:3000
//	   Access-Control-Request-Method: POST
//	   Access-Control-Request-Headers: Content-Type, Authorization
//
//	2. 服务器返回允许的配置
//	   Access-Control-Allow-Origin: http://localhost:3000
//	   Access-Control-Allow-Methods: POST
//	   Access-Control-Allow-Headers: Content-Type, Authorization
//
//	3. 浏览器确认允许后，发送实际请求
//	   POST /api/v1/chat
//	   ...
