package service

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"consiliumous-backend/internal/model"
)

// 统一错误定义。handler 层据此映射 HTTP 状态码。
var (
	ErrNotFound  = errors.New("资源不存在或无权限")
	ErrForbidden = errors.New("无权限执行该操作")
	ErrLastAdmin = errors.New("不能撤销项目最后一名管理员，请先指定接替者")
)

// ProjectService 承载项目、子项目、成员与角色的业务规则。
type ProjectService struct {
	DB *gorm.DB
}

func NewProjectService(db *gorm.DB) *ProjectService {
	return &ProjectService{DB: db}
}

type CreateProjectInput struct {
	Name        string     `json:"name" binding:"required"`
	Description string     `json:"description"`
	StartTime   *time.Time `json:"start_time"`
	EndTime     *time.Time `json:"end_time"`
}

type AddMemberInput struct {
	UserID uint   `json:"user_id" binding:"required"`
	Role   string `json:"role"` // 空则默认 member；可选 admin
}

type RevokeAdminInput struct {
	SuccessorID *uint // 撤销最后一名管理员时必须指定接替者
}

type MemberDTO struct {
	UserID uint   `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

// Create 创建项目：任何登录用户可创建，创建者自动成为管理员（PRD 3.2 / US-PROJ-01）。
func (s *ProjectService) Create(creatorID uint, in CreateProjectInput) (*model.Project, error) {
	p := model.Project{
		Name:        in.Name,
		Description: in.Description,
		Status:      model.ProjectActive,
		OwnerID:     creatorID,
		CreatedBy:   creatorID,
		StartTime:   in.StartTime,
		EndTime:     in.EndTime,
	}

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&p).Error; err != nil {
			return err
		}
		m := model.ProjectMember{ProjectID: p.ID, UserID: creatorID, Role: model.RoleAdmin}
		if err := tx.Create(&m).Error; err != nil {
			return err
		}
		return LogAudit(tx, creatorID, "project.create", "project", p.ID, "", toJSON(p))
	})
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ListForUser 返回当前用户参与（任意角色）的项目。系统管理员可见全部项目。
// 这是跨项目隔离的关键：非成员的项目根本不会出现在查询结果里（PRD 5.3.7）。
func (s *ProjectService) ListForUser(userID uint) ([]model.Project, error) {
	if sys, err := IsSystemAdmin(s.DB, userID); err != nil {
		return nil, err
	} else if sys {
		var all []model.Project
		err := s.DB.Order("id DESC").Find(&all).Error
		return all, err
	}
	var projects []model.Project
	err := s.DB.Model(&model.Project{}).
		Joins("JOIN project_members pm ON pm.project_id = projects.id").
		Where("pm.user_id = ?", userID).
		Order("projects.id DESC").
		Find(&projects).Error
	return projects, err
}

// GetForUser 返回项目详情；非成员得到 ErrNotFound（不区分"不存在"与"无权限"，PRD 5.4）。
// 系统管理员可查看任意项目。
func (s *ProjectService) GetForUser(projectID, userID uint) (*model.Project, error) {
	if sys, err := IsSystemAdmin(s.DB, userID); err != nil {
		return nil, err
	} else if sys {
		var p model.Project
		if err := s.DB.First(&p, projectID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrNotFound
			}
			return nil, err
		}
		return &p, nil
	}
	var p model.Project
	err := s.DB.Model(&model.Project{}).
		Joins("JOIN project_members pm ON pm.project_id = projects.id").
		Where("projects.id = ? AND pm.user_id = ?", projectID, userID).
		First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &p, err
}

// ListMembers 列出项目成员（需是项目成员）。
func (s *ProjectService) ListMembers(projectID, userID uint) ([]MemberDTO, error) {
	ok, err := s.isMember(s.DB, projectID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	var rows []MemberDTO
	err = s.DB.Model(&model.ProjectMember{}).
		Select("project_members.user_id, users.name, users.email, project_members.role").
		Joins("JOIN users ON users.id = project_members.user_id").
		Where("project_members.project_id = ?", projectID).
		Order("project_members.id").
		Scan(&rows).Error
	return rows, err
}

// AddMember 管理员添加成员（PRD US-PROJ-03）。
func (s *ProjectService) AddMember(actorID, projectID uint, in AddMemberInput) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		admin, err := s.isAdmin(tx, projectID, actorID)
		if err != nil {
			return err
		}
		if !admin {
			return ErrForbidden
		}

		var u model.User
		if err := tx.First(&u, in.UserID).Error; err != nil {
			return errors.New("用户不存在")
		}
		var count int64
		if err := tx.Model(&model.ProjectMember{}).
			Where("project_id = ? AND user_id = ?", projectID, in.UserID).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errors.New("该用户已是项目成员")
		}

		role := in.Role
		if role == "" {
			role = model.RoleMember
		}
		if role != model.RoleMember && role != model.RoleAdmin {
			return errors.New("非法角色")
		}

		m := model.ProjectMember{ProjectID: projectID, UserID: in.UserID, Role: role}
		if err := tx.Create(&m).Error; err != nil {
			return err
		}
		return LogAudit(tx, actorID, "project.member_add", "project", projectID, "", toJSON(m))
	})
}

// RemoveMember 移除成员；不能移除最后一名管理员（PRD US-PROJ-03 + 不变量 V4）。
func (s *ProjectService) RemoveMember(actorID, projectID, targetID uint) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		admin, err := s.isAdmin(tx, projectID, actorID)
		if err != nil {
			return err
		}
		if !admin {
			return ErrForbidden
		}

		var m model.ProjectMember
		if err := tx.Where("project_id = ? AND user_id = ?", projectID, targetID).First(&m).Error; err != nil {
			return ErrNotFound
		}
		if m.Role == model.RoleAdmin {
			admins, err := s.countAdmins(tx, projectID)
			if err != nil {
				return err
			}
			if admins <= 1 {
				return ErrLastAdmin
			}
		}

		before := toJSON(m)
		if err := tx.Delete(&m).Error; err != nil {
			return err
		}
		return LogAudit(tx, actorID, "project.member_remove", "project", projectID, before, "")
	})
}

// RevokeAdmin 撤销管理员。若目标是最后一名管理员，必须同时指定接替者，
// 一次性完成"任命接替者 + 撤销目标"（PRD US-PROJ-04）。
func (s *ProjectService) RevokeAdmin(actorID, projectID, targetID uint, in RevokeAdminInput) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		admin, err := s.isAdmin(tx, projectID, actorID)
		if err != nil {
			return err
		}
		if !admin {
			return ErrForbidden
		}

		var m model.ProjectMember
		if err := tx.Where("project_id = ? AND user_id = ?", projectID, targetID).First(&m).Error; err != nil {
			return ErrNotFound
		}
		if m.Role != model.RoleAdmin {
			return errors.New("该用户不是管理员")
		}

		admins, err := s.countAdmins(tx, projectID)
		if err != nil {
			return err
		}
		before := toJSON(m)

		// 最后一名管理员：必须指定接替者，且接替者须为项目成员。
		if admins <= 1 {
			if in.SuccessorID == nil || *in.SuccessorID == 0 {
				return ErrLastAdmin
			}
			var succ model.ProjectMember
			if err := tx.Where("project_id = ? AND user_id = ?", projectID, *in.SuccessorID).First(&succ).Error; err != nil {
				return errors.New("接替者不是本项目成员")
			}
			if err := tx.Model(&model.ProjectMember{}).Where("id = ?", succ.ID).Update("role", model.RoleAdmin).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.ProjectMember{}).Where("id = ?", m.ID).Update("role", model.RoleMember).Error; err != nil {
				return err
			}
			return LogAudit(tx, actorID, "project.admin_transfer", "project", projectID, before, toJSON(m)+"|"+toJSON(succ))
		}

		if err := tx.Model(&model.ProjectMember{}).Where("id = ?", m.ID).Update("role", model.RoleMember).Error; err != nil {
			return err
		}
		return LogAudit(tx, actorID, "project.admin_revoke", "project", projectID, before, toJSON(m))
	})
}

// Archive 归档/取消归档项目（管理员）。归档后项目只读，其下计划从个人视图隐藏（US-PROJ-05）。
func (s *ProjectService) Archive(actorID, projectID uint, archive bool) (*model.Project, error) {
	var p model.Project
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		ok, err := IsProjectAdminOrSys(tx, projectID, actorID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		if err := tx.First(&p, projectID).Error; err != nil {
			return ErrNotFound
		}
		before := toJSON(p)
		if archive {
			p.Status = model.ProjectArchived
		} else {
			p.Status = model.ProjectActive
		}
		if err := tx.Save(&p).Error; err != nil {
			return err
		}
		action := "project.archive"
		if !archive {
			action = "project.unarchive"
		}
		return LogAudit(tx, actorID, action, "project", projectID, before, toJSON(p))
	})
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Delete 永久删除项目（仅系统管理员，PRD US-PROJ-05）。级联删除子项目/Issue/计划/任务/成员/评论。
func (s *ProjectService) Delete(actorID, projectID uint) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		sys, err := IsSystemAdmin(tx, actorID)
		if err != nil {
			return err
		}
		if !sys {
			return ErrForbidden
		}
		var p model.Project
		if err := tx.First(&p, projectID).Error; err != nil {
			return ErrNotFound
		}
		before := toJSON(p)

		var planIDs []uint
		if err := tx.Model(&model.Plan{}).Where("project_id = ?", projectID).Pluck("id", &planIDs).Error; err != nil {
			return err
		}
		var issueIDs []uint
		if err := tx.Model(&model.Issue{}).Where("project_id = ?", projectID).Pluck("id", &issueIDs).Error; err != nil {
			return err
		}

		if len(planIDs) > 0 {
			if err := tx.Where("plan_id IN ?", planIDs).Delete(&model.Task{}).Error; err != nil {
				return err
			}
		}
		if len(issueIDs) > 0 {
			if err := tx.Where("issue_id IN ?", issueIDs).Delete(&model.Comment{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("project_id = ?", projectID).Delete(&model.Plan{}).Error; err != nil {
			return err
		}
		if err := tx.Where("project_id = ?", projectID).Delete(&model.Issue{}).Error; err != nil {
			return err
		}
		if err := tx.Where("project_id = ?", projectID).Delete(&model.SubProject{}).Error; err != nil {
			return err
		}
		if err := tx.Where("project_id = ?", projectID).Delete(&model.ProjectMember{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&p).Error; err != nil {
			return err
		}
		return LogAudit(tx, actorID, "project.delete", "project", projectID, before, "")
	})
}

// isAdmin 判断用户是否为该项目的管理员。
func (s *ProjectService) isAdmin(tx *gorm.DB, projectID, userID uint) (bool, error) {
	var count int64
	err := tx.Model(&model.ProjectMember{}).
		Where("project_id = ? AND user_id = ? AND role = ?", projectID, userID, model.RoleAdmin).
		Count(&count).Error
	return count > 0, err
}

// isMember 判断用户是否为该项目的成员（任意角色）。
func (s *ProjectService) isMember(tx *gorm.DB, projectID, userID uint) (bool, error) {
	var count int64
	err := tx.Model(&model.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Count(&count).Error
	return count > 0, err
}

// countAdmins 统计项目管理员数量。
func (s *ProjectService) countAdmins(tx *gorm.DB, projectID uint) (int64, error) {
	var count int64
	err := tx.Model(&model.ProjectMember{}).
		Where("project_id = ? AND role = ?", projectID, model.RoleAdmin).
		Count(&count).Error
	return count, err
}
