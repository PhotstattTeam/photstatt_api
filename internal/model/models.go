package model

import "time"

// 状态枚举的取值约定（字符串存库，便于阅读与扩展）
const (
	// 项目状态
	ProjectActive   = "active"   // 进行中
	ProjectArchived = "archived" // 已归档

	// 项目成员角色
	RoleAdmin  = "admin"
	RoleMember = "member"

	// Issue 类型
	IssueRequirement = "requirement" // 需求
	IssueBug         = "bug"         // 缺陷
	IssueOptimize    = "optimize"    // 优化
	IssueOther       = "other"       // 其他

	// Issue 状态
	IssuePending   = "pending"   // 待处理
	IssueScheduled = "scheduled" // 已排期
	IssueClosed    = "closed"    // 已关闭
	IssueRejected  = "rejected"  // 已拒绝

	// 计划来源（关键字段，决定归属与权限）
	PlanSourceProject  = "project"  // 项目排期
	PlanSourcePersonal = "personal" // 个人自建

	// 计划状态
	PlanNotStarted = "not_started" // 未开始
	PlanInProgress = "in_progress" // 进行中
	PlanDone       = "done"        // 已完成
	PlanCancelled  = "cancelled"   // 已取消

	// 任务状态
	TaskTodo       = "todo"        // 待办
	TaskInProgress = "in_progress" // 进行中
	TaskDone       = "done"        // 已完成
)

// User 用户。
type User struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Email         string    `gorm:"uniqueIndex;size:255;not null" json:"email"`
	Name          string    `gorm:"size:100;not null" json:"name"`
	PasswordHash  string    `gorm:"size:255;not null" json:"-"`
	IsSystemAdmin bool      `gorm:"not null;default:false" json:"is_system_admin"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Project 项目。任何登录用户可创建，创建者自动成为管理员。
type Project struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Name        string     `gorm:"size:200;not null" json:"name"`
	Description string     `gorm:"type:text" json:"description"`
	Status      string     `gorm:"size:20;not null;default:active" json:"status"`
	OwnerID     uint       `json:"owner_id"` // 负责人
	StartTime   *time.Time `json:"start_time"`
	EndTime     *time.Time `json:"end_time"`
	CreatedBy   uint       `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// SubProject 子项目。固定两级：ParentID 为 nil 是第一级，指向第一级的是第二级。
type SubProject struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ProjectID uint      `gorm:"index;not null" json:"project_id"`
	ParentID  *uint     `gorm:"index" json:"parent_id"` // 第二级指向第一级
	Name      string    `gorm:"size:200;not null" json:"name"`
	Sort      int       `gorm:"default:0" json:"sort"`
	OwnerID   *uint     `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProjectMember 项目成员。用户 × 项目 × 角色，权限按项目绑定。
type ProjectMember struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ProjectID uint      `gorm:"index:idx_project_user,unique;not null" json:"project_id"`
	UserID    uint      `gorm:"index:idx_project_user,unique;not null" json:"user_id"`
	Role      string    `gorm:"size:20;not null;default:member" json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// Issue 构成 Issue 池（bug / 需求 / 优化 / 其他）。
type Issue struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ProjectID    uint      `gorm:"index;not null" json:"project_id"`
	SubProjectID uint      `gorm:"index;not null" json:"sub_project_id"`
	Title        string    `gorm:"size:255;not null" json:"title"`
	Description  string    `gorm:"type:text" json:"description"`
	Type         string    `gorm:"size:20;not null" json:"type"`
	Priority     string    `gorm:"size:10;not null;default:P3" json:"priority"`
	Status       string    `gorm:"size:20;not null;default:pending" json:"status"`
	Resolution   string    `gorm:"size:255" json:"resolution"`  // 关闭/拒绝原因（PRD US-ISSUE-04）
	Tags         []string  `gorm:"serializer:json" json:"tags"` // 标签（PRD 4.2）
	CreatorID    uint      `json:"creator_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Plan 计划。项目计划（ProjectID/SubProjectID 非空）或个人计划（PlanGroupID 归属）。
type Plan struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ProjectID      *uint     `gorm:"index" json:"project_id"`     // 项目计划时非空
	SubProjectID   *uint     `gorm:"index" json:"sub_project_id"` // 项目计划时非空
	PlanGroupID    *uint     `gorm:"index" json:"plan_group_id"`  // 个人计划可归属的计划组
	Source         string    `gorm:"size:20;not null" json:"source"`
	Title          string    `gorm:"size:255;not null" json:"title"`
	Note           string    `gorm:"type:text" json:"note"`
	StartTime      time.Time `gorm:"not null" json:"start_time"`
	EndTime        time.Time `gorm:"not null" json:"end_time"`
	Progress       int       `gorm:"not null;default:0" json:"progress"`                                     // 0-100
	ProgressManual bool      `gorm:"not null;default:false" json:"progress_manual"`                          // 是否被人工覆盖
	IssueID        *uint     `gorm:"index" json:"issue_id"`                                                  // 来源 Issue（可空）
	OwnerID        uint      `gorm:"index;not null;uniqueIndex:idx_owner_client,priority:1" json:"owner_id"` // 负责人
	ClientID       *string   `gorm:"uniqueIndex:idx_owner_client,priority:2" json:"client_id"`               // 移动端幂等标识（同一用户内唯一）
	Status         string    `gorm:"size:20;not null;default:not_started" json:"status"`
	CreatedBy      uint      `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Overdue        bool      `gorm:"-" json:"overdue,omitempty"` // 派生状态：已延期（不存库，PRD 4.4）
}

// Task 计划任务。
type Task struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	PlanID     uint       `gorm:"index;not null" json:"plan_id"`
	Title      string     `gorm:"size:255;not null" json:"title"`
	AssigneeID uint       `json:"assignee_id"`
	Status     string     `gorm:"size:20;not null;default:todo" json:"status"`
	Sort       int        `gorm:"default:0" json:"sort"`
	DueAt      *time.Time `json:"due_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// PlanGroup 个人计划组（仅自建落库；"项目计划组"为运行时聚合，不建表）。
type PlanGroup struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	Sort      int       `gorm:"default:0" json:"sort"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AuditLog 审计日志：记录关键写操作的操作人、时间、前后值。
type AuditLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ActorID    uint      `gorm:"index" json:"actor_id"`
	Action     string    `gorm:"size:100;not null;index" json:"action"`
	TargetType string    `gorm:"size:50" json:"target_type"`
	TargetID   uint      `json:"target_id"`
	Before     string    `gorm:"type:text" json:"before"`
	After      string    `gorm:"type:text" json:"after"`
	CreatedAt  time.Time `json:"created_at"`
}

// UserPreference 用户偏好（键值对）。用于虚拟项目计划组的折叠/排序等，仅对本人有效（US-ME-03 / 5.3.5）。
type UserPreference struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index:idx_user_key,unique,priority:1;not null" json:"user_id"`
	Key       string    `gorm:"index:idx_user_key,unique,priority:2;size:100;not null" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Comment Issue 评论（PRD 权限矩阵表 B：评论 Issue）。
type Comment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	IssueID   uint      `gorm:"index;not null" json:"issue_id"`
	UserID    uint      `json:"user_id"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
