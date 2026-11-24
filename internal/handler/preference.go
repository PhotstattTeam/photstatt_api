package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"consiliumous-backend/internal/middleware"
)

// GetPreferences 返回当前用户的全部偏好（键值对）。
func (h *Handler) GetPreferences(c *gin.Context) {
	prefs, err := h.Prefs.GetAll(middleware.CurrentUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, prefs)
}

// SetPreference 设置某个偏好键（如虚拟项目计划组的折叠/排序）。
func (h *Handler) SetPreference(c *gin.Context) {
	me := middleware.CurrentUserID(c)
	var in struct {
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.Prefs.Set(me, c.Param("key"), in.Value); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
