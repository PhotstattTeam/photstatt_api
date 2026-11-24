package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"consiliumous-backend/internal/auth"
)

// contextUserID 是存入 gin.Context 的键，用于在下游取当前登录用户 ID。
const contextUserID = "user_id"

// Auth 解析 Authorization: Bearer <token>，把用户 ID 放进上下文。
// 挂在需要登录的路由组上。
func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "缺少令牌"})
			return
		}
		claims, err := auth.ParseToken(secret, strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "令牌无效或已过期"})
			return
		}
		c.Set(contextUserID, claims.UserID)
		c.Next()
	}
}

// CurrentUserID 从上下文取出当前登录用户 ID。
func CurrentUserID(c *gin.Context) uint {
	v, _ := c.Get(contextUserID)
	id, _ := v.(uint)
	return id
}
