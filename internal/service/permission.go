package service

import (
	"errors"

	"gorm.io/gorm"

	"consiliumous-backend/internal/model"
)

// 本文件集中存放权限判定助手，供各业务服务复用。
// PRD 5.1：RBAC（角色）+ 资源归属，所有校验在服务端执行。

// IsSystemAdmin 判断用户是否为全局系统管理员。
func IsSystemAdmin(db *gorm.DB, userID uint) (bool, error) {
	var u model.User
	err := db.Select("is_system_admin").First(&u, userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return u.IsSystemAdmin, nil
}

// IsProjectAdmin 判断用户是否为某项目的管理员。
func IsProjectAdmin(db *gorm.DB, projectID, userID uint) (bool, error) {
	var count int64
	err := db.Model(&model.ProjectMember{}).
		Where("project_id = ? AND user_id = ? AND role = ?", projectID, userID, model.RoleAdmin).
		Count(&count).Error
	return count > 0, err
}

// IsProjectMember 判断用户是否为某项目的成员（任意角色）。
func IsProjectMember(db *gorm.DB, projectID, userID uint) (bool, error) {
	var count int64
	err := db.Model(&model.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Count(&count).Error
	return count > 0, err
}

// IsProjectAdminOrSys 系统管理员或项目管理员。
func IsProjectAdminOrSys(db *gorm.DB, projectID, userID uint) (bool, error) {
	if sys, err := IsSystemAdmin(db, userID); err != nil {
		return false, err
	} else if sys {
		return true, nil
	}
	return IsProjectAdmin(db, projectID, userID)
}

// IsProjectMemberOrSys 系统管理员或项目成员。
func IsProjectMemberOrSys(db *gorm.DB, projectID, userID uint) (bool, error) {
	if sys, err := IsSystemAdmin(db, userID); err != nil {
		return false, err
	} else if sys {
		return true, nil
	}
	return IsProjectMember(db, projectID, userID)
}
