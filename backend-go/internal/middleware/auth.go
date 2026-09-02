package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	"lumiflybackend/internal/model"
)

// Auth JWT 认证。
type Auth struct {
	secret []byte
	db     *gorm.DB
}

// UserContext 当前登录用户上下文。
type UserContext struct {
	ID       uint
	Email    string
	Role     string
	IsActive bool
}

func NewAuth(secret string, db *gorm.DB) *Auth {
	return &Auth{secret: []byte(secret), db: db}
}

// CreateToken 签发 JWT（有效期 7 天）。
func (a *Auth) CreateToken(userID uint, email string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"userId": userID,
		"email":  email,
		"exp":    jwt.NewNumericDate(now.AddDate(0, 0, 7)),
		"iat":    jwt.NewNumericDate(now),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.secret)
}

func (a *Auth) parseToken(tokenStr string) (*jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return a.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrInvalidType
	}
	return &claims, nil
}

// RequireUser 需要登录的中间件。
// 每次请求校验账号仍存在且处于启用状态（被停用的账号会被立即拒绝）。
func (a *Auth) RequireUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := extractToken(c.GetHeader("Authorization"))
		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "请先登录"})
			return
		}
		claims, err := a.parseToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "登录已过期，请重新登录"})
			return
		}
		uidFloat, ok := (*claims)["userId"].(float64)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "凭证无效，请重新登录"})
			return
		}
		var u model.User
		if err := a.db.First(&u, uint(uidFloat)).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "账号不存在，请重新登录"})
			return
		}
		if !u.IsActive {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "账号已被停用，请联系管理员"})
			return
		}
		email, _ := (*claims)["email"].(string)
		c.Set("user", &UserContext{ID: u.ID, Email: email, Role: u.Role, IsActive: u.IsActive})
		c.Next()
	}
}

// RequireAdmin 需要管理员权限的中间件（须置于 RequireUser 之后）。
func (a *Auth) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get("user")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "请先登录"})
			return
		}
		cu := v.(*UserContext)
		if cu.Role != model.RoleAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"detail": "需要管理员权限"})
			return
		}
		c.Next()
	}
}

// SignResetToken 生成一次性密码重置令牌（有效期 30 分钟）。
func (a *Auth) SignResetToken(email string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"purpose": "reset_password",
		"email":   email,
		"iat":     now.Unix(),
		"exp":     now.Add(30 * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.secret)
}

// VerifyResetToken 校验重置令牌，返回其绑定的邮箱。
func (a *Auth) VerifyResetToken(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return a.secret, nil
	})
	if err != nil || !token.Valid {
		return "", err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["purpose"] != "reset_password" {
		return "", jwt.ErrInvalidType
	}
	email, ok := claims["email"].(string)
	if !ok || email == "" {
		return "", jwt.ErrInvalidType
	}
	return email, nil
}

func extractToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	}
	return strings.TrimSpace(authHeader)
}
