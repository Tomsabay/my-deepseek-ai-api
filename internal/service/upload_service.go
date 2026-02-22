// Package service 包含核心业务逻辑
// 这个文件实现了文件上传服务
package service

import (
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"ai-api/internal/model"
)

// ============================================
// UploadService 文件上传服务
// ============================================
type UploadService struct {
	// uploadDir 上传文件存储目录
	uploadDir string

	// maxFileSize 最大文件大小（字节）
	maxFileSize int64

	// allowedImageTypes 允许的图片类型
	allowedImageTypes map[string]bool

	// allowedDocTypes 允许的文档类型
	allowedDocTypes map[string]bool
}

// ============================================
// NewUploadService 创建上传服务实例
// ============================================
func NewUploadService(uploadDir string) *UploadService {
	// 确保上传目录存在
	os.MkdirAll(uploadDir, 0755)
	os.MkdirAll(filepath.Join(uploadDir, "images"), 0755)
	os.MkdirAll(filepath.Join(uploadDir, "documents"), 0755)

	return &UploadService{
		uploadDir:   uploadDir,
		maxFileSize: 20 * 1024 * 1024, // 20MB 限制
		allowedImageTypes: map[string]bool{
			"image/jpeg": true,
			"image/png":  true,
			"image/gif":  true,
			"image/webp": true,
		},
		allowedDocTypes: map[string]bool{
			"application/pdf":    true,
			"text/plain":         true,
			"text/markdown":      true,
			"text/csv":           true,
			"application/json":   true,
			"application/msword": true,
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		},
	}
}

// ============================================
// Upload 处理文件上传
// ============================================
func (s *UploadService) Upload(file multipart.File, header *multipart.FileHeader) (*model.Attachment, error) {
	// 1. 验证文件大小
	if header.Size > s.maxFileSize {
		return nil, fmt.Errorf("文件大小超过限制（最大 %dMB）", s.maxFileSize/1024/1024)
	}

	// 2. 获取 MIME 类型并验证
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// 3. 确定文件类型
	var fileType string
	var subDir string

	if s.allowedImageTypes[mimeType] {
		fileType = "image"
		subDir = "images"
	} else if s.allowedDocTypes[mimeType] {
		fileType = "document"
		subDir = "documents"
	} else {
		return nil, fmt.Errorf("不支持的文件类型: %s", mimeType)
	}

	// 4. 生成唯一文件名
	fileID := uuid.New().String()
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = s.guessExtension(mimeType)
	}
	newFilename := fileID + ext

	// 5. 保存文件
	savePath := filepath.Join(s.uploadDir, subDir, newFilename)
	dst, err := os.Create(savePath)
	if err != nil {
		return nil, fmt.Errorf("创建文件失败: %v", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return nil, fmt.Errorf("保存文件失败: %v", err)
	}

	// 6. 构建附件信息
	attachment := &model.Attachment{
		ID:        fileID,
		Filename:  header.Filename,
		Type:      fileType,
		MimeType:  mimeType,
		Size:      header.Size,
		URL:       fmt.Sprintf("/uploads/%s/%s", subDir, newFilename),
		CreatedAt: time.Now(),
	}

	return attachment, nil
}

// ============================================
// GetImageBase64 读取图片并转换为 base64
// ============================================
// 用于构建发送给 AI 的多模态消息
func (s *UploadService) GetImageBase64(attachmentURL string) (string, string, error) {
	// 构建完整路径
	relativePath := strings.TrimPrefix(attachmentURL, "/uploads/")
	fullPath := filepath.Join(s.uploadDir, relativePath)

	// 读取文件
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", "", fmt.Errorf("读取文件失败: %v", err)
	}

	// 获取 MIME 类型
	mimeType := s.guessMimeType(fullPath)

	// 转换为 base64
	base64Data := base64.StdEncoding.EncodeToString(data)

	return base64Data, mimeType, nil
}

// ============================================
// BuildImageDataURL 构建图片的 data URL
// ============================================
// 格式：data:image/jpeg;base64,...
func (s *UploadService) BuildImageDataURL(attachmentURL string) (string, error) {
	base64Data, mimeType, err := s.GetImageBase64(attachmentURL)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64Data), nil
}

// ============================================
// DeleteAttachment 删除附件
// ============================================
func (s *UploadService) DeleteAttachment(attachmentURL string) error {
	relativePath := strings.TrimPrefix(attachmentURL, "/uploads/")
	fullPath := filepath.Join(s.uploadDir, relativePath)
	return os.Remove(fullPath)
}

// ============================================
// 辅助函数
// ============================================

// guessExtension 根据 MIME 类型猜测文件扩展名
func (s *UploadService) guessExtension(mimeType string) string {
	extensions := map[string]string{
		"image/jpeg":       ".jpg",
		"image/png":        ".png",
		"image/gif":        ".gif",
		"image/webp":       ".webp",
		"application/pdf":  ".pdf",
		"text/plain":       ".txt",
		"text/markdown":    ".md",
		"text/csv":         ".csv",
		"application/json": ".json",
	}

	if ext, ok := extensions[mimeType]; ok {
		return ext
	}
	return ".bin"
}

// guessMimeType 根据文件扩展名猜测 MIME 类型
func (s *UploadService) guessMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	mimeTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".pdf":  "application/pdf",
		".txt":  "text/plain",
		".md":   "text/markdown",
		".csv":  "text/csv",
		".json": "application/json",
	}

	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

// ============================================
// 技术讲解：文件上传处理
// ============================================
//
// 【安全性考虑】
// 1. 文件大小限制 - 防止拒绝服务攻击
// 2. 文件类型白名单 - 只允许特定类型的文件
// 3. 随机文件名 - 防止文件名冲突和路径遍历攻击
// 4. 文件存储隔离 - 图片和文档分开存储
//
// 【Base64 编码】
// 将二进制图片转换为文本格式，方便在 JSON 中传输
// 格式：data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQABAAD...
//
// 【AI Vision API 支持】
// 大多数支持图片理解的 AI 模型都接受两种格式：
// 1. 图片 URL（需要 AI 服务能访问到该 URL）
// 2. Base64 编码的图片数据（更通用，不需要外部访问）
