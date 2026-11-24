package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"consiliumous-backend/internal/auth"
	"consiliumous-backend/internal/middleware"
	"consiliumous-backend/internal/service"
)

// Register 注册并自动登录，返回 token 与用户信息。
func (h *Handler) Register(c *gin.Context) {
	var in service.RegisterInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	u, err := h.Auth.Register(in)
	if err != nil {
		respondError(c, err)
		return
	}
	token, err := auth.GenerateToken(h.Auth.JWTSecret, u.ID, auth.DefaultTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"token": token, "user": u})
}

// Login 登录。
func (h *Handler) Login(c *gin.Context) {
	var in service.LoginInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	token, u, err := h.Auth.Login(in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "user": u})
}

// Me 返回当前登录用户信息。
func (h *Handler) Me(c *gin.Context) {
	u, err := h.Auth.GetByID(middleware.CurrentUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, u)
}

// SearchUsers 按姓名或邮箱搜索用户（供管理员加入项目）。
func (h *Handler) SearchUsers(c *gin.Context) {
	users, err := h.Auth.Search(c.Query("q"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, users)
}
