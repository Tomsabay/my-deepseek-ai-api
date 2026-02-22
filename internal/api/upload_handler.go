// Package api 包含 HTTP API 的处理逻辑
// 这个文件实现了文件上传相关的接口
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ai-api/internal/service"
)

// ============================================
// UploadHandler 文件上传处理器
// ============================================
type UploadHandler struct {
	uploadService *service.UploadService
}

// ============================================
// NewUploadHandler 创建上传处理器实例
// ============================================
func NewUploadHandler(uploadService *service.UploadService) *UploadHandler {
	return &UploadHandler{
		uploadService: uploadService,
	}
}

// ============================================
// RegisterRoutes 注册上传相关路由
// ============================================
func (h *UploadHandler) RegisterRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	{
		// POST /api/v1/upload - 上传文件
		// 支持图片和文档上传
		v1.POST("/upload", h.Upload)

		// POST /api/v1/upload/image - 专门的图片上传接口
		v1.POST("/upload/image", h.UploadImage)
	}
}

// ============================================
// Upload 通用文件上传接口
// ============================================
// @Summary 上传文件
// @Description 上传图片或文档文件，支持 jpg/png/gif/webp/pdf/txt/md/csv/json 格式
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "要上传的文件"
// @Success 200 {object} model.UploadResponse
// @Failure 400 {object} model.UploadResponse
// @Router /api/v1/upload [post]
func (h *UploadHandler) Upload(c *gin.Context) {
	// 1. 获取上传的文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "请选择要上传的文件",
		})
		return
	}
	defer file.Close()

	// 2. 调用上传服务处理文件
	attachment, err := h.uploadService.Upload(file, header)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 3. 返回上传结果
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"attachment": attachment,
	})
}

// ============================================
// UploadImage 专门的图片上传接口
// ============================================
// 只接受图片文件，提供更严格的验证
func (h *UploadHandler) UploadImage(c *gin.Context) {
	// 1. 获取上传的文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "请选择要上传的图片",
		})
		return
	}
	defer file.Close()

	// 2. 验证是否为图片类型
	contentType := header.Header.Get("Content-Type")
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}

	if !allowedTypes[contentType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "只支持 JPG、PNG、GIF、WebP 格式的图片",
		})
		return
	}

	// 3. 调用上传服务
	attachment, err := h.uploadService.Upload(file, header)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 4. 返回结果
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"attachment": attachment,
	})
}

// ============================================
// 技术讲解：Gin 文件上传
// ============================================
//
// 【multipart/form-data】
// 文件上传使用 multipart/form-data 编码格式，
// 这是 HTTP 标准中专门用于上传文件的格式。
//
// 【Gin 文件处理】
// c.Request.FormFile("file") 返回三个值：
// 1. file: 文件内容的 Reader
// 2. header: 文件头信息（文件名、大小、MIME类型）
// 3. err: 错误信息
//
// 【前端上传示例】
// const formData = new FormData();
// formData.append('file', fileInput.files[0]);
// fetch('/api/v1/upload', {
//     method: 'POST',
//     body: formData  // 不要设置 Content-Type，浏览器会自动设置
// });
