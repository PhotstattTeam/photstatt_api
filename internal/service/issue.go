package service

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"consiliumous-backend/internal/model"
)

// IssueService 承载 Issue 池业务（PRD US-ISSUE-01 ~ 04 + 权限矩阵表 B）。
type IssueService struct {
	DB *gorm.DB
}

func NewIssueService(db *gorm.DB) *IssueService {
	return &IssueService{DB: db}
}

type CreateIssueInput struct {
	SubProjectID uint     `json:"sub_project_id" binding:"required"`
	Title        string   `json:"title" binding:"required"`
	Description  string   `json:"description"`
	Type         string   `json:"type" binding:"required,oneof=requirement bug optimize other"`
	Priority     string   `json:"priority" binding:"required,oneof=P0 P1 P2 P3"`
	Tags         []string `json:"tags"`
}

type UpdateIssueInput struct {
	SubProjectID *uint     `json:"sub_project_id"`
	Title        *string   `json:"title"`
	Description  *string   `json:"description"`
	Type         *string   `json:"type"`
	Priority     *string   `json:"priority"`
	Tags         *[]string `json:"tags"`
}

type IssueStatusInput struct {
	Action string `json:"action" binding:"required,oneof=close reject"`
	Reason string `json:"reason" binding:"required"`
}

type IssueFilter struct {
	Status       string `form:"status"`
	Type         string `form:"type"`
	Priority     string `form:"priority"`
	SubProjectID uint   `form:"sub_project_id"`
	CreatorID    uint   `form:"creator_id"`
	Keyword      string `form:"keyword"`
	Sort         string `form:"sort"` // created | priority
}

// Create 新建 Issue（项目成员即可）。进入该子项目的 Issue 池，状态为"待处理"。
func (s *IssueService) Create(actorID, projectID uint, in CreateIssueInput) (*model.Issue, error) {
	var is *model.Issue
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		ok, err := IsProjectMemberOrSys(tx, projectID, actorID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		if err := validateSubProject(tx, projectID, in.SubProjectID); err != nil {
			return err
		}
		is = &model.Issue{
			ProjectID: projectID, SubProjectID: in.SubProjectID,
			Title: in.Title, Description: in.Description,
			Type: in.Type, Priority: in.Priority,
			Status: model.IssuePending, CreatorID: actorID,
			Tags: in.Tags,
		}
		if err := tx.Create(is).Error; err != nil {
			return err
		}
		return LogAudit(tx, actorID, "issue.create", "issue", is.ID, "", toJSON(is))
	})
	if err != nil {
		return nil, err
	}
	return is, nil
}

// List 列出 Issue 池。默认状态"待处理"；status=all 查看全部（PRD US-ISSUE-02/04）。
func (s *IssueService) List(projectID, userID uint, f IssueFilter) ([]model.Issue, error) {
	ok, err := IsProjectMemberOrSys(s.DB, projectID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}

	q := s.DB.Model(&model.Issue{}).Where("project_id = ?", projectID)

	status := f.Status
	if status == "" {
		status = model.IssuePending
	}
	if status != "all" {
		q = q.Where("status = ?", status)
	}
	if f.Type != "" {
		q = q.Where("type = ?", f.Type)
	}
	if f.Priority != "" {
		q = q.Where("priority = ?", f.Priority)
	}
	if f.SubProjectID != 0 {
		q = q.Where("sub_project_id = ?", f.SubProjectID)
	}
	if f.CreatorID != 0 {
		q = q.Where("creator_id = ?", f.CreatorID)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("title LIKE ? OR description LIKE ?", like, like)
	}

	switch f.Sort {
	case "priority":
		q = q.Order("priority ASC, created_at DESC")
	default:
		q = q.Order("created_at DESC")
	}

	var issues []model.Issue
	err = q.Find(&issues).Error
	return issues, err
}

// Get 查看单个 Issue。
func (s *IssueService) Get(projectID, issueID, userID uint) (*model.Issue, error) {
	ok, err := IsProjectMemberOrSys(s.DB, projectID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	var is model.Issue
	if err := s.DB.Where("id = ? AND project_id = ?", issueID, projectID).First(&is).Error; err != nil {
		return nil, ErrNotFound
	}
	return &is, nil
}

// Update 编辑 Issue。权限：系统管理员/项目管理员，或"本人提出且待处理"（PRD 表 B）。
func (s *IssueService) Update(actorID, projectID, issueID uint, in UpdateIssueInput) (*model.Issue, error) {
	var is model.Issue
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND project_id = ?", issueID, projectID).First(&is).Error; err != nil {
			return ErrNotFound
		}
		ok, err := s.canEdit(tx, projectID, actorID, is)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		if in.SubProjectID != nil {
			if err := validateSubProject(tx, projectID, *in.SubProjectID); err != nil {
				return err
			}
		}
		before := toJSON(is)
		if in.SubProjectID != nil {
			is.SubProjectID = *in.SubProjectID
		}
		if in.Title != nil {
			is.Title = *in.Title
		}
		if in.Description != nil {
			is.Description = *in.Description
		}
		if in.Type != nil {
			is.Type = *in.Type
		}
		if in.Priority != nil {
			is.Priority = *in.Priority
		}
		if in.Tags != nil {
			is.Tags = *in.Tags
		}
		if err := tx.Save(&is).Error; err != nil {
			return err
		}
		return LogAudit(tx, actorID, "issue.update", "issue", issueID, before, toJSON(is))
	})
	if err != nil {
		return nil, err
	}
	return &is, nil
}

// Delete 删除 Issue。权限同编辑。
func (s *IssueService) Delete(actorID, projectID, issueID uint) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		var is model.Issue
		if err := tx.Where("id = ? AND project_id = ?", issueID, projectID).First(&is).Error; err != nil {
			return ErrNotFound
		}
		ok, err := s.canEdit(tx, projectID, actorID, is)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		before := toJSON(is)
		if err := tx.Delete(&is).Error; err != nil {
			return err
		}
		return LogAudit(tx, actorID, "issue.delete", "issue", issueID, before, "")
	})
}

// ChangeStatus 关闭或拒绝 Issue（仅系统管理员/项目管理员，PRD US-ISSUE-04）。
func (s *IssueService) ChangeStatus(actorID, projectID, issueID uint, in IssueStatusInput) (*model.Issue, error) {
	var is model.Issue
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		ok, err := IsProjectAdminOrSys(tx, projectID, actorID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		if err := tx.Where("id = ? AND project_id = ?", issueID, projectID).First(&is).Error; err != nil {
			return ErrNotFound
		}
		before := toJSON(is)
		switch in.Action {
		case "close":
			is.Status = model.IssueClosed
		case "reject":
			is.Status = model.IssueRejected
		}
		is.Resolution = in.Reason
		if err := tx.Save(&is).Error; err != nil {
			return err
		}
		return LogAudit(tx, actorID, "issue."+in.Action, "issue", issueID, before, toJSON(is))
	})
	if err != nil {
		return nil, err
	}
	return &is, nil
}

// AddComment 评论 Issue（项目成员即可，PRD 表 B）。
func (s *IssueService) AddComment(actorID, projectID, issueID uint, content string) (*model.Comment, error) {
	ok, err := IsProjectMemberOrSys(s.DB, projectID, actorID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	var is model.Issue
	if err := s.DB.Where("id = ? AND project_id = ?", issueID, projectID).First(&is).Error; err != nil {
		return nil, ErrNotFound
	}
	cm := model.Comment{IssueID: issueID, UserID: actorID, Content: content}
	if err := s.DB.Create(&cm).Error; err != nil {
		return nil, err
	}
	return &cm, nil
}

// ListComments 列出某 Issue 的评论。
func (s *IssueService) ListComments(projectID, issueID, userID uint) ([]model.Comment, error) {
	ok, err := IsProjectMemberOrSys(s.DB, projectID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	var is model.Issue
	if err := s.DB.Where("id = ? AND project_id = ?", issueID, projectID).First(&is).Error; err != nil {
		return nil, ErrNotFound
	}
	var cms []model.Comment
	err = s.DB.Where("issue_id = ?", issueID).Order("id ASC").Find(&cms).Error
	return cms, err
}

// canEdit 编辑/删除 Issue 的权限：系统管理员/项目管理员，或"本人提出且待处理"。
func (s *IssueService) canEdit(tx *gorm.DB, projectID, actorID uint, is model.Issue) (bool, error) {
	if sys, err := IsSystemAdmin(tx, actorID); err != nil {
		return false, err
	} else if sys {
		return true, nil
	}
	if admin, err := IsProjectAdmin(tx, projectID, actorID); err != nil {
		return false, err
	} else if admin {
		return true, nil
	}
	// 本人提出且未排期（待处理）才可编辑
	return is.CreatorID == actorID && is.Status == model.IssuePending, nil
}

// validateSubProject 校验子项目属于该项目。
func validateSubProject(tx *gorm.DB, projectID, subProjectID uint) error {
	var count int64
	if err := tx.Model(&model.SubProject{}).
		Where("id = ? AND project_id = ?", subProjectID, projectID).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("子项目不存在或不属于该项目")
	}
	return nil
}
