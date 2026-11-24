package service

import (
	"errors"

	"gorm.io/gorm"

	"consiliumous-backend/internal/model"
)

// SubProjectService 承载子项目业务（PRD US-PROJ-02：固定两级）。
type SubProjectService struct {
	DB *gorm.DB
}

func NewSubProjectService(db *gorm.DB) *SubProjectService {
	return &SubProjectService{DB: db}
}

// SubProjectInput 创建/更新子项目。Update 采用全量更新语义（前端总是传完整对象）。
type SubProjectInput struct {
	Name     string `json:"name" binding:"required"`
	ParentID *uint  `json:"parent_id"` // 空=第一级；非空=挂在某个第一级下
	Sort     int    `json:"sort"`
	OwnerID  *uint  `json:"owner_id"`
}

// List 列出项目下的子项目（成员即可见，供创建 Issue 时选择归属）。
func (s *SubProjectService) List(projectID, userID uint) ([]model.SubProject, error) {
	ok, err := IsProjectMemberOrSys(s.DB, projectID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	var sps []model.SubProject
	err = s.DB.Where("project_id = ?", projectID).Order("sort ASC, id ASC").Find(&sps).Error
	return sps, err
}

// Create 创建子项目（管理员）。层级固定两级。
func (s *SubProjectService) Create(actorID, projectID uint, in SubProjectInput) (*model.SubProject, error) {
	var sp *model.SubProject
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		ok, err := IsProjectAdminOrSys(tx, projectID, actorID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		if err := validateParentLevel(tx, projectID, in.ParentID); err != nil {
			return err
		}
		sp = &model.SubProject{
			ProjectID: projectID, ParentID: in.ParentID,
			Name: in.Name, Sort: in.Sort, OwnerID: in.OwnerID,
		}
		if err := tx.Create(sp).Error; err != nil {
			return err
		}
		return LogAudit(tx, actorID, "subproject.create", "subproject", sp.ID, "", toJSON(sp))
	})
	if err != nil {
		return nil, err
	}
	return sp, nil
}

// Update 修改子项目（管理员，全量更新）。
func (s *SubProjectService) Update(actorID, projectID, spID uint, in SubProjectInput) (*model.SubProject, error) {
	var sp model.SubProject
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		ok, err := IsProjectAdminOrSys(tx, projectID, actorID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		if err := tx.Where("id = ? AND project_id = ?", spID, projectID).First(&sp).Error; err != nil {
			return ErrNotFound
		}
		if in.ParentID != nil && *in.ParentID == sp.ID {
			return errors.New("不能把子项目设为自身的父级")
		}
		if err := validateParentLevel(tx, projectID, in.ParentID); err != nil {
			return err
		}
		before := toJSON(sp)
		sp.Name = in.Name
		sp.ParentID = in.ParentID
		sp.Sort = in.Sort
		sp.OwnerID = in.OwnerID
		if err := tx.Save(&sp).Error; err != nil {
			return err
		}
		return LogAudit(tx, actorID, "subproject.update", "subproject", sp.ID, before, toJSON(sp))
	})
	if err != nil {
		return nil, err
	}
	return &sp, nil
}

// Delete 删除子项目（管理员）。其下存在 Issue 时禁止删除，避免产生孤儿数据。
func (s *SubProjectService) Delete(actorID, projectID, spID uint) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		ok, err := IsProjectAdminOrSys(tx, projectID, actorID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		var sp model.SubProject
		if err := tx.Where("id = ? AND project_id = ?", spID, projectID).First(&sp).Error; err != nil {
			return ErrNotFound
		}
		var issues int64
		if err := tx.Model(&model.Issue{}).Where("sub_project_id = ?", spID).Count(&issues).Error; err != nil {
			return err
		}
		if issues > 0 {
			return errors.New("该子项目下存在 Issue，请先处理后再删除")
		}
		before := toJSON(sp)
		if err := tx.Delete(&sp).Error; err != nil {
			return err
		}
		return LogAudit(tx, actorID, "subproject.delete", "subproject", spID, before, "")
	})
}

// validateParentLevel 校验两级结构：ParentID 为空是第一级；否则其父级必须是第一级（父级的父级为空）。
func validateParentLevel(tx *gorm.DB, projectID uint, parentID *uint) error {
	if parentID == nil {
		return nil
	}
	var parent model.SubProject
	if err := tx.Where("id = ? AND project_id = ?", *parentID, projectID).First(&parent).Error; err != nil {
		return errors.New("父级子项目不存在")
	}
	if parent.ParentID != nil {
		return errors.New("只允许两级子项目，不能在第二级下再建子项目")
	}
	return nil
}
