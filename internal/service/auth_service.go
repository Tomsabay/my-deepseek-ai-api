// Package service 包含用户认证相关的业务逻辑
package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"ai-api/internal/database"
	"ai-api/internal/model"
)

// ============================================
// 认证相关错误
// ============================================
var (
	ErrUserNotFound      = errors.New("用户不存在")
	ErrUserAlreadyExists = errors.New("用户名已存在")
	ErrInvalidPassword   = errors.New("密码错误")
	ErrInvalidToken      = errors.New("无效的 Token")
	ErrTokenExpired      = errors.New("Token 已过期")
)

// ============================================
// AuthService 认证服务
// ============================================
type AuthService struct {
	jwtSecret     []byte        // JWT 签名密钥
	tokenExpiry   time.Duration // Token 有效期
	refreshExpiry time.Duration // Refresh Token 有效期
}

// ============================================
// NewAuthService 创建认证服务实例
// ============================================
func NewAuthService(jwtSecret string) *AuthService {
	return &AuthService{
		jwtSecret:     []byte(jwtSecret),
		tokenExpiry:   24 * time.Hour,     // Access Token 24 小时有效
		refreshExpiry: 7 * 24 * time.Hour, // Refresh Token 7 天有效
	}
}

// ============================================
// Register 用户注册
// ============================================
// 参数：
//   - username: 用户名
//   - password: 明文密码
//   - nickname: 昵称（可选）
//
// 返回：
//   - *model.User: 创建的用户
//   - error: 错误信息
func (s *AuthService) Register(username, password, nickname string) (*model.User, error) {
	// 检查用户名是否已存在
	var existingUser model.User
	result := database.DB.Where("username = ?", username).First(&existingUser)
	if result.Error == nil {
		return nil, ErrUserAlreadyExists
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	}

	// 加密密码
	// bcrypt 是目前最推荐的密码哈希算法
	// Cost 参数越高越安全，但也越慢（推荐 10-12）
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 如果没有提供昵称，使用用户名
	if nickname == "" {
		nickname = username
	}

	// 创建用户
	user := &model.User{
		Username:     username,
		PasswordHash: string(hashedPassword),
		Nickname:     nickname,
	}

	if err := database.DB.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

// ============================================
// Login 用户登录
// ============================================
// 参数：
//   - username: 用户名
//   - password: 明文密码
//
// 返回：
//   - *model.User: 用户信息
//   - string: Access Token
//   - string: Refresh Token
//   - error: 错误信息
func (s *AuthService) Login(username, password string) (*model.User, string, string, error) {
	// 查找用户
	var user model.User
	result := database.DB.Where("username = ?", username).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, "", "", ErrUserNotFound
	}
	if result.Error != nil {
		return nil, "", "", result.Error
	}

	// 验证密码
	// bcrypt.CompareHashAndPassword 会自动处理 salt
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, "", "", ErrInvalidPassword
	}

	// 生成 Token
	accessToken, err := s.generateToken(user.ID, s.tokenExpiry)
	if err != nil {
		return nil, "", "", err
	}

	refreshToken, err := s.generateToken(user.ID, s.refreshExpiry)
	if err != nil {
		return nil, "", "", err
	}

	return &user, accessToken, refreshToken, nil
}

// ============================================
// ValidateToken 验证 Token
// ============================================
// 返回值：
//   - uint: 用户 ID
//   - error: 错误信息
func (s *AuthService) ValidateToken(tokenString string) (uint, error) {
	// 解析 Token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return 0, ErrTokenExpired
		}
		return 0, ErrInvalidToken
	}

	// 提取 Claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID := uint(claims["user_id"].(float64))
		return userID, nil
	}

	return 0, ErrInvalidToken
}

// ============================================
// GetUserByID 根据 ID 获取用户
// ============================================
func (s *AuthService) GetUserByID(userID uint) (*model.User, error) {
	var user model.User
	result := database.DB.First(&user, userID)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	return &user, result.Error
}

// ============================================
// generateToken 生成 JWT Token
// ============================================
func (s *AuthService) generateToken(userID uint, expiry time.Duration) (string, error) {
	// 创建 Claims（Token 载荷）
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(expiry).Unix(), // 过期时间
		"iat":     time.Now().Unix(),             // 签发时间
	}

	// 创建 Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 签名并返回
	return token.SignedString(s.jwtSecret)
}

// ============================================
// 技术讲解：JWT (JSON Web Token)
// ============================================
//
// JWT 是一种无状态的认证方式，Token 由三部分组成：
// Header.Payload.Signature
//
// 1. Header（头部）：
//    {"alg": "HS256", "typ": "JWT"}
//    指定签名算法
//
// 2. Payload（载荷）：
//    {"user_id": 1, "exp": 1234567890}
//    包含用户信息和过期时间
//
// 3. Signature（签名）：
//    HMACSHA256(base64(header) + "." + base64(payload), secret)
//    使用密钥签名，防止篡改
//
// 优点：
// - 无状态：服务端不需要存储 session
// - 可扩展：可以在 Payload 中添加自定义数据
// - 跨域友好：适合前后端分离架构
//
// 缺点：
// - 无法主动失效：Token 一旦签发，只能等待过期
// - 体积较大：比 session ID 更长
//
// 使用 Refresh Token 的原因：
// - Access Token 有效期短（如 1 天），安全性高
// - Refresh Token 有效期长（如 7 天），用于刷新 Access Token
// - 这样即使 Access Token 泄露，也只有短时间的风险
