package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"consiliumous-backend/internal/middleware"
	"consiliumous-backend/internal/service"
)

// ListTasks 列出计划下的任务。
func (h *Handler) ListTasks(c *gin.Context) {
	pid, ok1 := parsePathID(c, "id")
	planID, ok2 := parsePathID(c, "planId")
	if !ok1 || !ok2 {
		return
	}
	tasks, err := h.Task.List(pid, planID, middleware.CurrentUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, tasks)
}

// CreateTask 新增任务。
func (h *Handler) CreateTask(c *gin.Context) {
	pid, ok1 := parsePathID(c, "id")
	planID, ok2 := parsePathID(c, "planId")
	if !ok1 || !ok2 {
		return
	}
	var in service.CreateTaskInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t, err := h.Task.Create(middleware.CurrentUserID(c), pid, planID, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, t)
}

// UpdateTask 编辑任务。
func (h *Handler) UpdateTask(c *gin.Context) {
	pid, ok1 := parsePathID(c, "id")
	planID, ok2 := parsePathID(c, "planId")
	taskID, ok3 := parsePathID(c, "taskId")
	if !ok1 || !ok2 || !ok3 {
		return
	}
	var in service.UpdateTaskInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t, err := h.Task.Update(middleware.CurrentUserID(c), pid, planID, taskID, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, t)
}

// DeleteTask 删除任务。
func (h *Handler) DeleteTask(c *gin.Context) {
	pid, ok1 := parsePathID(c, "id")
	planID, ok2 := parsePathID(c, "planId")
	taskID, ok3 := parsePathID(c, "taskId")
	if !ok1 || !ok2 || !ok3 {
		return
	}
	if err := h.Task.Delete(middleware.CurrentUserID(c), pid, planID, taskID); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// UpdateTaskStatus 更新任务状态。
func (h *Handler) UpdateTaskStatus(c *gin.Context) {
	pid, ok1 := parsePathID(c, "id")
	planID, ok2 := parsePathID(c, "planId")
	taskID, ok3 := parsePathID(c, "taskId")
	if !ok1 || !ok2 || !ok3 {
		return
	}
	var in service.TaskStatusInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t, err := h.Task.UpdateStatus(middleware.CurrentUserID(c), pid, planID, taskID, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, t)
}

// RestorePlanProgress 恢复计划进度自动计算。
func (h *Handler) RestorePlanProgress(c *gin.Context) {
	pid, ok1 := parsePathID(c, "id")
	planID, ok2 := parsePathID(c, "planId")
	if !ok1 || !ok2 {
		return
	}
	p, err := h.Plan.RestoreAutoProgress(middleware.CurrentUserID(c), pid, planID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}
